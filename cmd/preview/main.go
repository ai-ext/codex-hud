package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ds/codex-hud/internal/config"
	gitpkg "github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/parser"
	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

// findBestSession walks sessionsDir and returns the most recently modified
// .jsonl file that has at least minSize bytes (to skip near-empty sessions).
// Falls back to the largest file if none meet the recency+size criteria.
func findBestSession(sessionsDir string, minSize int64) (string, error) {
	type fileEntry struct {
		path    string
		size    int64
		modTime int64
	}

	var entries []fileEntry

	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			entries = append(entries, fileEntry{
				path:    path,
				size:    info.Size(),
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

	// Pick the most recent file that meets the minimum size.
	for _, e := range entries {
		if e.size >= minSize {
			return e.path, nil
		}
	}

	// Fallback: pick the largest file overall.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].size > entries[j].size
	})
	return entries[0].path, nil
}

func main() {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".codex", "sessions")

	sessionFile, err := findBestSession(sessionsDir, 10*1024) // 10KB minimum
	if err != nil {
		fmt.Println("No session files found:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Using session: %s\n", sessionFile)

	f, err := os.Open(sessionFile)
	if err != nil {
		fmt.Println("Cannot open session:", err)
		return
	}
	defer f.Close()
	events, _ := parser.ParseLines(f)

	s := state.New()
	for _, e := range events {
		switch e.Type {
		case "session_meta":
			if m, err := e.AsSessionMeta(); err == nil {
				s.ApplySessionMeta(m)
			}

		case "turn_context":
			if tc, err := e.AsTurnContext(); err == nil {
				s.ApplyTurnContext(tc)
				s.IncrementTurn()
			}

		case "event_msg":
			subtype, err := e.EventMsgType()
			if err != nil {
				continue
			}
			switch subtype {
			case "token_count":
				if tc, err := e.AsTokenCount(); err == nil {
					s.ApplyTokenCount(tc)
				}
			}

		case "response_item":
			// Peek at type/subtype to determine how to handle.
			var env struct {
				Type    string `json:"type"`
				Subtype string `json:"subtype"`
			}
			if err := json.Unmarshal(e.Payload, &env); err != nil {
				continue
			}
			itemType := env.Subtype
			if itemType == "" {
				itemType = env.Type
			}
			switch itemType {
			case "function_call":
				if fc, err := e.AsFunctionCall(); err == nil {
					s.ApplyFunctionCall(fc)
				}
			case "function_call_output":
				var output struct {
					CallID string `json:"call_id"`
				}
				if err := json.Unmarshal(e.Payload, &output); err == nil && output.CallID != "" {
					s.CompleteFunctionCall(output.CallID)
				}
			}
		}
	}

	// Override StartTime for realistic duration display.
	s.StartTime = time.Now().Add(-20 * time.Minute)

	cfg := config.Default()
	gitStatus, _ := gitpkg.GetStatus(s.CWD)

	lines := make(chan string)
	model := tui.NewModel(cfg, lines)
	model.State = s
	model.GitStatus = gitStatus
	// Use 80+ width to trigger side-by-side layout for tokens/session.
	width := 80
	if len(os.Args) > 1 && os.Args[1] == "narrow" {
		width = 60
	}
	model.Width = width
	model.Height = 30
	model.Waiting = false

	fmt.Println(model.View())
}
