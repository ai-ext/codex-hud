// Package watcher provides file watching utilities for tailing Codex session logs.
package watcher

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ds/codex-hud/internal/parser"
	"github.com/fsnotify/fsnotify"
)

// debug controls verbose logging for troubleshooting session detection.
// Set via SetDebug.
var debug bool

// SetDebug enables or disables verbose debug logging for the watcher package.
func SetDebug(enabled bool) { debug = enabled }

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf("[watcher] "+format, args...)
	}
}

// FindLatestSession walks sessionsDir recursively, finds all .jsonl files,
// and returns the path of the most recently modified one.
// It returns an error if no .jsonl files are found.
func FindLatestSession(sessionsDir string) (string, error) {
	type fileEntry struct {
		path    string
		modTime int64
	}

	var entries []fileEntry

	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			info, err := d.Info()
			if err != nil {
				return err
			}
			entries = append(entries, fileEntry{
				path:    path,
				modTime: info.ModTime().UnixNano(),
			})
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking sessions dir: %w", err)
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no .jsonl files found in %s", sessionsDir)
	}

	// Sort by modification time, newest first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime > entries[j].modTime
	})

	return entries[0].path, nil
}

// ReadExistingLines reads all current lines from a file and returns them as a
// slice. This is used to pre-populate state before starting the TUI, avoiding
// the "jumpy" startup where lines are processed one by one.
func ReadExistingLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\n\r")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// TailFile opens the file at path, reads all existing lines and sends each
// non-empty line to the lines channel. It then watches for new appends via
// fsnotify and sends new lines as they appear. TailFile blocks until the stop
// channel is closed, then returns nil. It uses bufio.Reader for streaming reads.
func TailFile(path string, lines chan<- string, stop <-chan struct{}) error {
	debugf("TailFile: opening %s", path)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// Read all existing content line by line.
	var partial string
	if err := readLines(reader, lines, &partial); err != nil {
		return err
	}
	debugf("TailFile: initial read done for %s (partial=%q)", path, partial)

	return tailFromReader(f, reader, path, lines, stop)
}

// TailFileFromEnd is like TailFile but seeks to the end of the file first,
// only capturing newly appended lines. Use this when existing content has
// already been pre-loaded via ReadExistingLines to avoid duplicate processing.
func TailFileFromEnd(path string, lines chan<- string, stop <-chan struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Seek to end — only new appends will be read.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek %s: %w", path, err)
	}

	reader := bufio.NewReader(f)
	return tailFromReader(f, reader, path, lines, stop)
}

// tailFromReader watches the file for new appends and sends new lines through
// the channel. Shared implementation for TailFile and TailFileFromEnd.
func tailFromReader(f *os.File, reader *bufio.Reader, path string, lines chan<- string, stop <-chan struct{}) error {
	// Set up fsnotify watcher for new appends (best-effort — poll is the
	// fallback for platforms where fsnotify is unreliable).
	var fsEvents <-chan fsnotify.Event
	var fsErrors <-chan error
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		defer watcher.Close()
		if err := watcher.Add(path); err == nil {
			fsEvents = watcher.Events
			fsErrors = watcher.Errors
		}
	}

	// Poll as fallback in case fsnotify misses Write events (common on Windows).
	pollTicker := time.NewTicker(200 * time.Millisecond)
	defer pollTicker.Stop()

	var partial string
	for {
		select {
		case <-stop:
			return nil
		case <-pollTicker.C:
			if err := readLines(reader, lines, &partial); err != nil {
				return err
			}
		case event, ok := <-fsEvents:
			if !ok {
				fsEvents = nil
				continue
			}
			if event.Has(fsnotify.Write) {
				if err := readLines(reader, lines, &partial); err != nil {
					return err
				}
			}
		case _, ok := <-fsErrors:
			if !ok {
				fsErrors = nil
				continue
			}
			// Non-fatal: fall back to polling.
		}
	}
}

// readLines reads all available complete lines from reader and sends non-empty
// ones to the lines channel. Partial lines (without a trailing newline) are
// stored in *partial so they can be prepended to the next read.
func readLines(reader *bufio.Reader, lines chan<- string, partial *string) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Partial line (no trailing newline yet). Save it so the
				// next call can prepend it once more data is available.
				if line != "" {
					*partial += line
				}
				return nil
			}
			return fmt.Errorf("read line: %w", err)
		}
		// Prepend any previously buffered partial data.
		if *partial != "" {
			line = *partial + line
			*partial = ""
		}
		line = strings.TrimRight(line, "\n\r")
		if line != "" {
			lines <- line
		}
	}
}

// FindLatestRateLimits scans the most recent session files (up to maxFiles)
// looking for the last non-null rate_limits in token_count events. Returns nil
// if no rate limits are found.
func FindLatestRateLimits(sessionsDir string, maxFiles int) *parser.RateLimits {
	type fileEntry struct {
		path    string
		modTime int64
	}

	var entries []fileEntry
	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			entries = append(entries, fileEntry{path: path, modTime: info.ModTime().UnixNano()})
		}
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime > entries[j].modTime
	})

	for i, e := range entries {
		if i >= maxFiles {
			break
		}
		if rl := scanFileForRateLimits(e.path); rl != nil {
			return rl
		}
	}
	return nil
}

// scanFileForRateLimits reads a JSONL file backwards (last lines first via
// buffer) and returns the most recent non-null rate_limits.
func scanFileForRateLimits(path string) *parser.RateLimits {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Read all lines, check from the end for the most recent rate_limits.
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Scan from newest to oldest.
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if !strings.Contains(line, "rate_limits") || strings.Contains(line, `"rate_limits":null`) {
			continue
		}
		evt, err := parser.ParseLine(line)
		if err != nil || evt.Type != "event_msg" {
			continue
		}
		subtype, err := evt.EventMsgType()
		if err != nil || subtype != "token_count" {
			continue
		}
		tc, err := evt.AsTokenCount()
		if err != nil || tc.RateLimits == nil {
			continue
		}
		return tc.RateLimits
	}
	return nil
}

// WatchForNewSession watches sessionsDir recursively via fsnotify. When a new
// .jsonl file is created, it starts TailFile on it. It also watches for new
// subdirectories (Codex creates date-based dirs). It blocks until stop is closed.
// A periodic poll runs as a fallback in case fsnotify misses events (common on
// Windows).
//
// If initialFile is non-empty, WatchForNewSession assumes that file is already
// being tailed elsewhere and won't start a duplicate TailFile for it. It will
// still detect NEWER session files (e.g. from /resume).
func WatchForNewSession(sessionsDir string, lines chan<- string, stop <-chan struct{}, minModTime time.Time, initialFile ...string) {
	debugf("WatchForNewSession: dir=%s minModTime=%v initial=%v", sessionsDir, minModTime, initialFile)

	// Set up fsnotify (best-effort — poll always runs as fallback).
	var fsEvents <-chan fsnotify.Event
	var fsErrors <-chan error
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		defer watcher.Close()
		_ = watcher.Add(sessionsDir)
		_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				_ = watcher.Add(path)
			}
			return nil
		})
		fsEvents = watcher.Events
		fsErrors = watcher.Errors
		debugf("WatchForNewSession: fsnotify initialized")
	} else {
		debugf("WatchForNewSession: fsnotify failed (%v), using poll-only", err)
	}

	// Track the stop channel for the currently tailed file so we can stop
	// tailing when a newer session appears.
	var currentStop chan struct{}
	var currentFile string

	// If an initial file was provided, record it so we don't start a
	// duplicate TailFile.
	if len(initialFile) > 0 && initialFile[0] != "" {
		currentFile = initialFile[0]
		debugf("WatchForNewSession: initial file set to %s", currentFile)
	}

	// startTailing switches to tailing a new session file.
	startTailing := func(path string) {
		if path == currentFile {
			return // already tailing this file
		}
		debugf("WatchForNewSession: switching from %q to %s", currentFile, path)
		if currentStop != nil {
			close(currentStop)
		}
		currentStop = make(chan struct{})
		currentFile = path
		go TailFile(path, lines, currentStop)
	}

	// Periodic poll as fallback for missed fsnotify events (especially on Windows).
	pollTicker := time.NewTicker(300 * time.Millisecond)
	defer pollTicker.Stop()

	for {
		select {
		case <-stop:
			if currentStop != nil {
				close(currentStop)
			}
			return
		case <-pollTicker.C:
			// Poll for the latest .jsonl file in case fsnotify missed it.
			latest, err := FindLatestSession(sessionsDir)
			if err == nil && latest != "" {
				// In fresh mode, skip files older than minModTime to
				// avoid loading stale session data.
				if !minModTime.IsZero() {
					if info, e := os.Stat(latest); e == nil {
						if info.ModTime().Before(minModTime) {
							continue
						}
					}
				}
				startTailing(latest)
			}
		case event, ok := <-fsEvents:
			if !ok {
				fsEvents = nil
				continue
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err != nil {
					continue
				}
				if info.IsDir() {
					// New subdirectory: start watching it too.
					if watcher != nil {
						_ = watcher.Add(event.Name)
					}
					// Scan for .jsonl files that may have been created
					// between the directory creation and our watcher.Add.
					entries, _ := os.ReadDir(event.Name)
					for _, e := range entries {
						if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
							startTailing(filepath.Join(event.Name, e.Name()))
						}
					}
					continue
				}
				if strings.HasSuffix(event.Name, ".jsonl") {
					debugf("WatchForNewSession: fsnotify Create %s", event.Name)
					startTailing(event.Name)
				}
			}
		case _, ok := <-fsErrors:
			if !ok {
				fsErrors = nil
				continue
			}
			// Non-fatal: polling continues as fallback.
		}
	}
}
