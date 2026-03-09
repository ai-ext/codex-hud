package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ds/codex-hud/internal/parser"
)

func TestStateWithRealSession(t *testing.T) {
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

	events, errs := parser.ParseLines(f)
	t.Logf("Parsed %d events with %d errors from %s", len(events), len(errs), sessionFile)

	if len(events) == 0 {
		t.Fatal("expected at least 1 event")
	}

	// Build state by processing all events
	sess := New()

	turnContextSeen := false
	tokenCountApplied := 0
	functionCallApplied := 0

	for _, ev := range events {
		switch ev.Type {
		case "session_meta":
			meta, err := ev.AsSessionMeta()
			if err != nil {
				t.Logf("skipping session_meta: %v", err)
				continue
			}
			sess.ApplySessionMeta(meta)

		case "turn_context":
			tc, err := ev.AsTurnContext()
			if err != nil {
				t.Logf("skipping turn_context: %v", err)
				continue
			}
			sess.ApplyTurnContext(tc)
			sess.IncrementTurn()
			turnContextSeen = true

		case "event_msg":
			// Try parsing as token_count; only apply if info has
			// meaningful data (the first token_count in a session
			// may have a null info field).
			tc, err := ev.AsTokenCount()
			if err == nil && tc != nil && tc.Info.TotalTokenUsage.InputTokens > 0 {
				sess.ApplyTokenCount(tc)
				tokenCountApplied++
			}

		case "response_item":
			fc, err := ev.AsFunctionCall()
			if err == nil && fc != nil && fc.Name != "" {
				sess.ApplyFunctionCall(fc)
				functionCallApplied++
			}
		}
	}

	t.Logf("Applied: turn_context=%v, token_counts=%d, function_calls=%d",
		turnContextSeen, tokenCountApplied, functionCallApplied)

	// Verify session metadata was set
	if sess.SessionID == "" {
		t.Error("SessionID is empty after applying events")
	} else {
		t.Logf("SessionID: %s", sess.SessionID)
	}

	if sess.CLIVersion == "" {
		t.Error("CLIVersion is empty")
	} else {
		t.Logf("CLIVersion: %s", sess.CLIVersion)
	}

	if sess.CWD == "" {
		t.Error("CWD is empty")
	} else {
		t.Logf("CWD: %s", sess.CWD)
	}

	if sess.StartTime.IsZero() {
		t.Error("StartTime is zero")
	} else {
		t.Logf("StartTime: %s", sess.StartTime)
	}

	// Verify model info if turn_context was seen
	if turnContextSeen {
		if sess.Model == "" {
			t.Error("Model is empty after turn_context")
		} else {
			t.Logf("Model: %s", sess.Model)
		}
		t.Logf("ApprovalPolicy: %s, SandboxType: %s, ReasoningEffort: %s",
			sess.ApprovalPolicy, sess.SandboxType, sess.ReasoningEffort)
	}

	// Verify token counts if any were applied
	if tokenCountApplied > 0 {
		if sess.TotalInputTokens == 0 {
			t.Error("TotalInputTokens is 0 after applying token counts")
		}
		if sess.ContextWindowSize == 0 {
			t.Error("ContextWindowSize is 0 after applying token counts")
		}

		pct := sess.ContextPercent()
		t.Logf("Tokens: in=%d, cached=%d, out=%d, reasoning=%d",
			sess.TotalInputTokens, sess.TotalCachedTokens,
			sess.TotalOutputTokens, sess.TotalReasonTokens)
		t.Logf("Context: %d / %d (%.1f%%)",
			sess.ContextUsedTokens, sess.ContextWindowSize, pct)

		// Note: ContextPercent can exceed 100% because TotalTokens in
		// TotalTokenUsage is cumulative across the entire session, not
		// just the current context window fill. This is expected.
		if pct < 0 {
			t.Errorf("ContextPercent = %.2f, expected non-negative", pct)
		}
	}

	// Verify rate limits if present
	if sess.HasRateLimits {
		t.Logf("Rate limits: primary=%.1f%%, secondary=%.1f%%",
			sess.PrimaryRatePercent, sess.SecondaryRatePercent)
	}

	// Verify turn count
	t.Logf("TurnCount: %d", sess.TurnCount)
	if turnContextSeen && sess.TurnCount == 0 {
		t.Error("TurnCount is 0 but turn_context events were seen")
	}

	// Verify tool counts
	if functionCallApplied > 0 {
		t.Logf("ToolCounts: %v", sess.ToolCounts)
		totalTools := 0
		for _, count := range sess.ToolCounts {
			totalTools += count
		}
		if totalTools != functionCallApplied {
			t.Errorf("total tool count %d != function calls applied %d",
				totalTools, functionCallApplied)
		}
	}

	// Log final state summary
	t.Logf("--- Final State Summary ---")
	t.Logf("  Session: %s (v%s)", sess.SessionID, sess.CLIVersion)
	t.Logf("  Model: %s, Provider: %s", sess.Model, sess.ModelProvider)
	t.Logf("  Turns: %d, Tools used: %d", sess.TurnCount, functionCallApplied)
	if tokenCountApplied > 0 {
		t.Logf("  Context: %.1f%% (%d/%d tokens)",
			sess.ContextPercent(), sess.ContextUsedTokens, sess.ContextWindowSize)
	}
}
