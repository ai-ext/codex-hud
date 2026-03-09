package state

import (
	"math"
	"testing"

	"github.com/ds/codex-hud/internal/parser"
)

// ---------------------------------------------------------------------------
// TestNewSession
// ---------------------------------------------------------------------------

func TestNewSession(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", s.SessionID)
	}
	if s.TurnCount != 0 {
		t.Errorf("TurnCount = %d, want 0", s.TurnCount)
	}
	if s.ToolCounts == nil {
		t.Fatal("ToolCounts map is nil, want initialized")
	}
	if len(s.ToolCounts) != 0 {
		t.Errorf("ToolCounts has %d entries, want 0", len(s.ToolCounts))
	}
	if s.ActiveTools == nil {
		t.Fatal("ActiveTools slice is nil, want initialized")
	}
	if len(s.ActiveTools) != 0 {
		t.Errorf("ActiveTools has %d entries, want 0", len(s.ActiveTools))
	}
}

// ---------------------------------------------------------------------------
// TestApplySessionMeta
// ---------------------------------------------------------------------------

func TestApplySessionMeta(t *testing.T) {
	s := New()
	meta := &parser.SessionMeta{
		ID:            "sess_abc123",
		Timestamp:     "2025-06-02T10:00:00.000Z",
		CWD:           "/home/user/project",
		CLIVersion:    "0.1.2025060200",
		ModelProvider: "openai",
	}
	s.ApplySessionMeta(meta)

	if s.SessionID != "sess_abc123" {
		t.Errorf("SessionID = %q, want %q", s.SessionID, "sess_abc123")
	}
	if s.CLIVersion != "0.1.2025060200" {
		t.Errorf("CLIVersion = %q, want %q", s.CLIVersion, "0.1.2025060200")
	}
	if s.CWD != "/home/user/project" {
		t.Errorf("CWD = %q, want %q", s.CWD, "/home/user/project")
	}
	if s.ModelProvider != "openai" {
		t.Errorf("ModelProvider = %q, want %q", s.ModelProvider, "openai")
	}
	if s.StartTime.IsZero() {
		t.Error("StartTime is zero, want parsed timestamp")
	}
	if s.StartTime.Year() != 2025 {
		t.Errorf("StartTime.Year() = %d, want 2025", s.StartTime.Year())
	}
}

// ---------------------------------------------------------------------------
// TestApplyTurnContext
// ---------------------------------------------------------------------------

func TestApplyTurnContext(t *testing.T) {
	s := New()
	tc := &parser.TurnContext{
		Model:          "o4-mini",
		ApprovalPolicy: "auto-edit",
		SandboxPolicy:  parser.SandboxPolicy{Type: "docker"},
		CollaborationMode: parser.CollaborationMode{
			Settings: parser.CollaborationModeSettings{
				ReasoningEffort: "medium",
			},
		},
	}
	s.ApplyTurnContext(tc)

	if s.Model != "o4-mini" {
		t.Errorf("Model = %q, want %q", s.Model, "o4-mini")
	}
	if s.ReasoningEffort != "medium" {
		t.Errorf("ReasoningEffort = %q, want %q", s.ReasoningEffort, "medium")
	}
	if s.ApprovalPolicy != "auto-edit" {
		t.Errorf("ApprovalPolicy = %q, want %q", s.ApprovalPolicy, "auto-edit")
	}
	if s.SandboxType != "docker" {
		t.Errorf("SandboxType = %q, want %q", s.SandboxType, "docker")
	}
}

// ---------------------------------------------------------------------------
// TestApplyTokenCount
// ---------------------------------------------------------------------------

func TestApplyTokenCount(t *testing.T) {
	s := New()
	tc := &parser.TokenCount{
		Info: parser.TokenInfo{
			TotalTokenUsage: parser.TokenUsage{
				InputTokens:           5000,
				CachedInputTokens:     1200,
				OutputTokens:          800,
				ReasoningOutputTokens: 200,
				TotalTokens:           6000,
			},
			ModelContextWindow: 128000,
		},
	}
	s.ApplyTokenCount(tc)

	if s.TotalInputTokens != 5000 {
		t.Errorf("TotalInputTokens = %d, want 5000", s.TotalInputTokens)
	}
	if s.TotalCachedTokens != 1200 {
		t.Errorf("TotalCachedTokens = %d, want 1200", s.TotalCachedTokens)
	}
	if s.TotalOutputTokens != 800 {
		t.Errorf("TotalOutputTokens = %d, want 800", s.TotalOutputTokens)
	}
	if s.TotalReasonTokens != 200 {
		t.Errorf("TotalReasonTokens = %d, want 200", s.TotalReasonTokens)
	}
	if s.ContextWindowSize != 128000 {
		t.Errorf("ContextWindowSize = %d, want 128000", s.ContextWindowSize)
	}
	if s.ContextUsedTokens != 6000 {
		t.Errorf("ContextUsedTokens = %d, want 6000", s.ContextUsedTokens)
	}
	if s.HasRateLimits {
		t.Error("HasRateLimits = true, want false when no rate limits")
	}

	// ContextPercent: 6000/128000 * 100 ~= 4.6875
	pct := s.ContextPercent()
	expected := float64(6000) / float64(128000) * 100.0
	if math.Abs(pct-expected) > 0.01 {
		t.Errorf("ContextPercent() = %.4f, want ~%.4f", pct, expected)
	}
}

func TestApplyTokenCountWithRateLimits(t *testing.T) {
	s := New()
	tc := &parser.TokenCount{
		Info: parser.TokenInfo{
			TotalTokenUsage: parser.TokenUsage{
				InputTokens:           35000,
				CachedInputTokens:     10000,
				OutputTokens:          5000,
				ReasoningOutputTokens: 1000,
				TotalTokens:           40000,
			},
			ModelContextWindow: 128000,
		},
		RateLimits: &parser.RateLimits{
			Primary: parser.RateLimit{
				UsedPercent:   42.5,
				WindowMinutes: 1,
				ResetsAt:      1717322460,
			},
			Secondary: parser.RateLimit{
				UsedPercent:   10.0,
				WindowMinutes: 60,
				ResetsAt:      1717325400,
			},
		},
	}
	s.ApplyTokenCount(tc)

	if !s.HasRateLimits {
		t.Error("HasRateLimits = false, want true")
	}
	if s.PrimaryRatePercent != 42.5 {
		t.Errorf("PrimaryRatePercent = %f, want 42.5", s.PrimaryRatePercent)
	}
	if s.PrimaryResetsAt != 1717322460 {
		t.Errorf("PrimaryResetsAt = %d, want 1717322460", s.PrimaryResetsAt)
	}
	if s.SecondaryRatePercent != 10.0 {
		t.Errorf("SecondaryRatePercent = %f, want 10.0", s.SecondaryRatePercent)
	}
	if s.SecondaryResetsAt != 1717325400 {
		t.Errorf("SecondaryResetsAt = %d, want 1717325400", s.SecondaryResetsAt)
	}

	// ContextPercent: 40000/128000 * 100 ~= 31.25 -- not ~27% but the test below
	// uses exact values. The spec says ~27% which we can achieve with different
	// token values. Let's just verify the math is correct.
	pct := s.ContextPercent()
	expected := float64(40000) / float64(128000) * 100.0
	if math.Abs(pct-expected) > 0.01 {
		t.Errorf("ContextPercent() = %.4f, want ~%.4f", pct, expected)
	}
}

// ---------------------------------------------------------------------------
// TestContextPercentApprox27
// ---------------------------------------------------------------------------

func TestContextPercentApprox27(t *testing.T) {
	s := New()
	tc := &parser.TokenCount{
		Info: parser.TokenInfo{
			TotalTokenUsage: parser.TokenUsage{
				TotalTokens: 34560,
			},
			ModelContextWindow: 128000,
		},
	}
	s.ApplyTokenCount(tc)
	pct := s.ContextPercent()
	// 34560 / 128000 * 100 = 27.0
	if math.Abs(pct-27.0) > 0.1 {
		t.Errorf("ContextPercent() = %.4f, want ~27.0", pct)
	}
}

// ---------------------------------------------------------------------------
// TestTrackToolCalls
// ---------------------------------------------------------------------------

func TestTrackToolCalls(t *testing.T) {
	s := New()

	// Add 2 exec_command calls and 1 apply_patch call
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_001",
	})
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_002",
	})
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "apply_patch",
		CallID: "call_003",
	})

	if s.ToolCounts["exec_command"] != 2 {
		t.Errorf("ToolCounts[exec_command] = %d, want 2", s.ToolCounts["exec_command"])
	}
	if s.ToolCounts["apply_patch"] != 1 {
		t.Errorf("ToolCounts[apply_patch] = %d, want 1", s.ToolCounts["apply_patch"])
	}
	if len(s.ActiveTools) != 3 {
		t.Fatalf("ActiveTools has %d entries, want 3", len(s.ActiveTools))
	}

	// Verify active tool entries
	found := map[string]bool{}
	for _, at := range s.ActiveTools {
		found[at.CallID] = true
		if at.StartAt.IsZero() {
			t.Errorf("ActiveTool %q has zero StartAt", at.CallID)
		}
	}
	for _, id := range []string{"call_001", "call_002", "call_003"} {
		if !found[id] {
			t.Errorf("ActiveTools missing call %q", id)
		}
	}
}

// ---------------------------------------------------------------------------
// TestCompleteToolCall
// ---------------------------------------------------------------------------

func TestCompleteToolCall(t *testing.T) {
	s := New()

	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_001",
	})
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "apply_patch",
		CallID: "call_002",
	})

	if len(s.ActiveTools) != 2 {
		t.Fatalf("ActiveTools has %d entries before complete, want 2", len(s.ActiveTools))
	}

	s.CompleteFunctionCall("call_001")

	if len(s.ActiveTools) != 1 {
		t.Fatalf("ActiveTools has %d entries after complete, want 1", len(s.ActiveTools))
	}
	if s.ActiveTools[0].CallID != "call_002" {
		t.Errorf("remaining ActiveTools[0].CallID = %q, want %q", s.ActiveTools[0].CallID, "call_002")
	}

	// ToolCounts should be unchanged
	if s.ToolCounts["exec_command"] != 1 {
		t.Errorf("ToolCounts[exec_command] = %d, want 1 (unchanged)", s.ToolCounts["exec_command"])
	}
}

// ---------------------------------------------------------------------------
// TestIncrementTurns
// ---------------------------------------------------------------------------

func TestIncrementTurns(t *testing.T) {
	s := New()

	if s.TurnCount != 0 {
		t.Errorf("TurnCount = %d, want 0", s.TurnCount)
	}

	s.IncrementTurn()
	if s.TurnCount != 1 {
		t.Errorf("TurnCount = %d after first increment, want 1", s.TurnCount)
	}

	s.IncrementTurn()
	s.IncrementTurn()
	if s.TurnCount != 3 {
		t.Errorf("TurnCount = %d after 3 increments, want 3", s.TurnCount)
	}
}
