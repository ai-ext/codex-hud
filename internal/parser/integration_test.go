package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRealSessionFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home dir")
	}

	sessionsDir := filepath.Join(home, ".codex", "sessions")

	// Find any .jsonl file
	var sessionFile string
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".jsonl" && sessionFile == "" {
			sessionFile = path
		}
		return nil
	})

	if sessionFile == "" {
		t.Skip("no Codex session files found")
	}

	f, err := os.Open(sessionFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	events, errs := ParseLines(f)
	t.Logf("Parsed %d events with %d errors from %s", len(events), len(errs), sessionFile)

	if len(events) == 0 {
		t.Error("expected at least 1 event")
	}

	// First event should be session_meta
	if events[0].Type != "session_meta" {
		t.Errorf("first event should be session_meta, got %s", events[0].Type)
	}

	// Verify session_meta can be decoded
	meta, err := events[0].AsSessionMeta()
	if err != nil {
		t.Errorf("AsSessionMeta: %v", err)
	} else {
		t.Logf("Session ID: %s, CLI version: %s, provider: %s",
			meta.ID, meta.CLIVersion, meta.ModelProvider)
	}

	// Count event types
	counts := make(map[string]int)
	for _, e := range events {
		counts[e.Type]++
	}
	t.Logf("Event type distribution: %v", counts)

	// Should have token_count events within event_msg
	hasTokenCount := false
	for _, e := range events {
		if e.Type == "event_msg" {
			// Try parsing as token_count directly; in real data the
			// discriminator field may be "type" rather than "subtype",
			// so we attempt deserialization and check for non-zero info.
			tc, err := e.AsTokenCount()
			if err == nil && tc != nil && tc.Info.TotalTokenUsage.InputTokens > 0 {
				hasTokenCount = true
				t.Logf("Token count: in=%d, cached=%d, out=%d, reasoning=%d, total=%d, context=%d",
					tc.Info.TotalTokenUsage.InputTokens,
					tc.Info.TotalTokenUsage.CachedInputTokens,
					tc.Info.TotalTokenUsage.OutputTokens,
					tc.Info.TotalTokenUsage.ReasoningOutputTokens,
					tc.Info.TotalTokenUsage.TotalTokens,
					tc.Info.ModelContextWindow)
				break
			}
		}
	}
	if !hasTokenCount {
		t.Log("Warning: no token_count events with non-zero info found")
	}

	// Check for turn_context events
	hasTurnContext := false
	for _, e := range events {
		if e.Type == "turn_context" {
			tc, err := e.AsTurnContext()
			if err == nil && tc != nil {
				hasTurnContext = true
				t.Logf("Turn context: model=%s, approval=%s, sandbox=%s",
					tc.Model, tc.ApprovalPolicy, tc.SandboxPolicy.Type)
				break
			}
		}
	}
	if !hasTurnContext {
		t.Log("Warning: no turn_context events found")
	}

	// Check for function_call events in response_item
	fcCount := 0
	toolNames := make(map[string]int)
	for _, e := range events {
		if e.Type == "response_item" {
			fc, err := e.AsFunctionCall()
			if err == nil && fc != nil && fc.Name != "" {
				fcCount++
				toolNames[fc.Name]++
			}
		}
	}
	if fcCount > 0 {
		t.Logf("Function calls: %d total, tools: %v", fcCount, toolNames)
	} else {
		t.Log("Warning: no function_call events found")
	}
}
