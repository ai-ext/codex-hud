// Package watcher provides file watching utilities for tailing Codex session logs.
package watcher

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"
)

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

// TailFile opens the file at path, reads all existing lines and sends each
// non-empty line to the lines channel. It then watches for new appends via
// fsnotify and sends new lines as they appear. TailFile blocks until the stop
// channel is closed, then returns nil. It uses bufio.Reader for streaming reads.
func TailFile(path string, lines chan<- string, stop <-chan struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// Read all existing content line by line.
	if err := readLines(reader, lines); err != nil {
		return err
	}

	// Set up fsnotify watcher for new appends.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		return fmt.Errorf("watch %s: %w", path, err)
	}

	for {
		select {
		case <-stop:
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) {
				if err := readLines(reader, lines); err != nil {
					return err
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// readLines reads all available complete lines from reader and sends non-empty
// ones to the lines channel.
func readLines(reader *bufio.Reader, lines chan<- string) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No more data available right now. If we got a partial
				// line (no trailing newline), ignore it for now; it will
				// be completed on the next write event.
				return nil
			}
			return fmt.Errorf("read line: %w", err)
		}
		line = strings.TrimRight(line, "\n\r")
		if line != "" {
			lines <- line
		}
	}
}

// WatchForNewSession watches sessionsDir recursively via fsnotify. When a new
// .jsonl file is created, it starts TailFile on it. It also watches for new
// subdirectories (Codex creates date-based dirs). It blocks until stop is closed.
func WatchForNewSession(sessionsDir string, lines chan<- string, stop <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// Add the root sessions directory.
	_ = watcher.Add(sessionsDir)

	// Also add any existing subdirectories.
	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = watcher.Add(path)
		}
		return nil
	})

	// Track the stop channel for the currently tailed file so we can stop
	// tailing when a newer session appears.
	var currentStop chan struct{}

	for {
		select {
		case <-stop:
			if currentStop != nil {
				close(currentStop)
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err != nil {
					continue
				}
				if info.IsDir() {
					// New subdirectory: start watching it too.
					_ = watcher.Add(event.Name)
					continue
				}
				if strings.HasSuffix(event.Name, ".jsonl") {
					// Stop tailing the previous session if any.
					if currentStop != nil {
						close(currentStop)
					}
					currentStop = make(chan struct{})
					go TailFile(event.Name, lines, currentStop)
				}
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
