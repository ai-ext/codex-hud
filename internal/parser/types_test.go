package parser

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Realistic JSONL snippets captured from actual Codex CLI sessions
// ---------------------------------------------------------------------------

const sampleSessionMeta = `{
  "timestamp": "2025-06-02T10:00:00.000Z",
  "type": "session_meta",
  "payload": {
    "id": "sess_abc123",
    "timestamp": "2025-06-02T10:00:00.000Z",
    "cwd": "/home/user/project",
    "originator": "codex-cli",
    "cli_version": "0.1.2025060200",
    "source": "terminal",
    "model_provider": "openai"
  }
}`

const sampleTurnContext = `{
  "timestamp": "2025-06-02T10:00:01.000Z",
  "type": "turn_context",
  "payload": {
    "turn_id": "turn_001",
    "cwd": "/home/user/project",
    "model": "o4-mini",
    "personality": "You are a helpful coding assistant.",
    "approval_policy": "auto-edit",
    "sandbox_policy": {
      "type": "docker"
    },
    "collaboration_mode": {
      "mode": "pair",
      "settings": {
        "model": "o4-mini",
        "reasoning_effort": "medium"
      }
    }
  }
}`

const sampleTokenCount = `{
  "timestamp": "2025-06-02T10:00:05.000Z",
  "type": "event_msg",
  "payload": {
    "subtype": "token_count",
    "info": {
      "total_token_usage": {
        "input_tokens": 5000,
        "cached_input_tokens": 1200,
        "output_tokens": 800,
        "reasoning_output_tokens": 200,
        "total_tokens": 6000
      },
      "last_token_usage": {
        "input_tokens": 1000,
        "cached_input_tokens": 300,
        "output_tokens": 150,
        "reasoning_output_tokens": 50,
        "total_tokens": 1200
      },
      "model_context_window": 128000
    },
    "rate_limits": {
      "primary": {
        "used_percent": 42.5,
        "window_minutes": 1,
        "resets_at": 1717322460
      },
      "secondary": {
        "used_percent": 10.0,
        "window_minutes": 60,
        "resets_at": 1717325400
      }
    }
  }
}`

const sampleTokenCountNoRateLimits = `{
  "timestamp": "2025-06-02T10:00:06.000Z",
  "type": "event_msg",
  "payload": {
    "subtype": "token_count",
    "info": {
      "total_token_usage": {
        "input_tokens": 100,
        "cached_input_tokens": 0,
        "output_tokens": 50,
        "reasoning_output_tokens": 0,
        "total_tokens": 150
      },
      "last_token_usage": {
        "input_tokens": 100,
        "cached_input_tokens": 0,
        "output_tokens": 50,
        "reasoning_output_tokens": 0,
        "total_tokens": 150
      },
      "model_context_window": 128000
    }
  }
}`

const sampleFunctionCall = `{
  "timestamp": "2025-06-02T10:00:03.000Z",
  "type": "response_item",
  "payload": {
    "subtype": "function_call",
    "name": "shell",
    "arguments": "{\"cmd\":\"ls -la\"}",
    "call_id": "call_xyz789"
  }
}`

// ---------------------------------------------------------------------------
// Event envelope
// ---------------------------------------------------------------------------

func TestEventUnmarshal(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleSessionMeta), &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if ev.Timestamp != "2025-06-02T10:00:00.000Z" {
		t.Errorf("timestamp = %q, want %q", ev.Timestamp, "2025-06-02T10:00:00.000Z")
	}
	if ev.Type != "session_meta" {
		t.Errorf("type = %q, want %q", ev.Type, "session_meta")
	}
	if ev.Payload == nil {
		t.Fatal("payload is nil")
	}
}

// ---------------------------------------------------------------------------
// EventMsgType
// ---------------------------------------------------------------------------

func TestEventMsgType(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleTokenCount), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	subtype, err := ev.EventMsgType()
	if err != nil {
		t.Fatalf("EventMsgType error: %v", err)
	}
	if subtype != "token_count" {
		t.Errorf("subtype = %q, want %q", subtype, "token_count")
	}
}

func TestEventMsgType_WrongType(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleSessionMeta), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	_, err := ev.EventMsgType()
	if err == nil {
		t.Fatal("expected error for non-event_msg type")
	}
}

// ---------------------------------------------------------------------------
// AsSessionMeta
// ---------------------------------------------------------------------------

func TestAsSessionMeta(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleSessionMeta), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	meta, err := ev.AsSessionMeta()
	if err != nil {
		t.Fatalf("AsSessionMeta: %v", err)
	}

	if meta.ID != "sess_abc123" {
		t.Errorf("ID = %q, want %q", meta.ID, "sess_abc123")
	}
	if meta.Timestamp != "2025-06-02T10:00:00.000Z" {
		t.Errorf("Timestamp = %q", meta.Timestamp)
	}
	if meta.CWD != "/home/user/project" {
		t.Errorf("CWD = %q", meta.CWD)
	}
	if meta.Originator != "codex-cli" {
		t.Errorf("Originator = %q", meta.Originator)
	}
	if meta.CLIVersion != "0.1.2025060200" {
		t.Errorf("CLIVersion = %q", meta.CLIVersion)
	}
	if meta.Source != "terminal" {
		t.Errorf("Source = %q", meta.Source)
	}
	if meta.ModelProvider != "openai" {
		t.Errorf("ModelProvider = %q", meta.ModelProvider)
	}
}

// ---------------------------------------------------------------------------
// AsTurnContext
// ---------------------------------------------------------------------------

func TestAsTurnContext(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleTurnContext), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	tc, err := ev.AsTurnContext()
	if err != nil {
		t.Fatalf("AsTurnContext: %v", err)
	}

	if tc.TurnID != "turn_001" {
		t.Errorf("TurnID = %q", tc.TurnID)
	}
	if tc.CWD != "/home/user/project" {
		t.Errorf("CWD = %q", tc.CWD)
	}
	if tc.Model != "o4-mini" {
		t.Errorf("Model = %q", tc.Model)
	}
	if tc.Personality != "You are a helpful coding assistant." {
		t.Errorf("Personality = %q", tc.Personality)
	}
	if tc.ApprovalPolicy != "auto-edit" {
		t.Errorf("ApprovalPolicy = %q", tc.ApprovalPolicy)
	}
	if tc.SandboxPolicy.Type != "docker" {
		t.Errorf("SandboxPolicy.Type = %q", tc.SandboxPolicy.Type)
	}
	if tc.CollaborationMode.Mode != "pair" {
		t.Errorf("CollaborationMode.Mode = %q", tc.CollaborationMode.Mode)
	}
	if tc.CollaborationMode.Settings.Model != "o4-mini" {
		t.Errorf("CollaborationMode.Settings.Model = %q", tc.CollaborationMode.Settings.Model)
	}
	if tc.CollaborationMode.Settings.ReasoningEffort != "medium" {
		t.Errorf("CollaborationMode.Settings.ReasoningEffort = %q", tc.CollaborationMode.Settings.ReasoningEffort)
	}
}

// ---------------------------------------------------------------------------
// AsTokenCount
// ---------------------------------------------------------------------------

func TestAsTokenCount(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleTokenCount), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	tc, err := ev.AsTokenCount()
	if err != nil {
		t.Fatalf("AsTokenCount: %v", err)
	}

	// total_token_usage
	if tc.Info.TotalTokenUsage.InputTokens != 5000 {
		t.Errorf("TotalTokenUsage.InputTokens = %d", tc.Info.TotalTokenUsage.InputTokens)
	}
	if tc.Info.TotalTokenUsage.CachedInputTokens != 1200 {
		t.Errorf("TotalTokenUsage.CachedInputTokens = %d", tc.Info.TotalTokenUsage.CachedInputTokens)
	}
	if tc.Info.TotalTokenUsage.OutputTokens != 800 {
		t.Errorf("TotalTokenUsage.OutputTokens = %d", tc.Info.TotalTokenUsage.OutputTokens)
	}
	if tc.Info.TotalTokenUsage.ReasoningOutputTokens != 200 {
		t.Errorf("TotalTokenUsage.ReasoningOutputTokens = %d", tc.Info.TotalTokenUsage.ReasoningOutputTokens)
	}
	if tc.Info.TotalTokenUsage.TotalTokens != 6000 {
		t.Errorf("TotalTokenUsage.TotalTokens = %d", tc.Info.TotalTokenUsage.TotalTokens)
	}

	// last_token_usage
	if tc.Info.LastTokenUsage.InputTokens != 1000 {
		t.Errorf("LastTokenUsage.InputTokens = %d", tc.Info.LastTokenUsage.InputTokens)
	}
	if tc.Info.LastTokenUsage.CachedInputTokens != 300 {
		t.Errorf("LastTokenUsage.CachedInputTokens = %d", tc.Info.LastTokenUsage.CachedInputTokens)
	}
	if tc.Info.LastTokenUsage.OutputTokens != 150 {
		t.Errorf("LastTokenUsage.OutputTokens = %d", tc.Info.LastTokenUsage.OutputTokens)
	}
	if tc.Info.LastTokenUsage.ReasoningOutputTokens != 50 {
		t.Errorf("LastTokenUsage.ReasoningOutputTokens = %d", tc.Info.LastTokenUsage.ReasoningOutputTokens)
	}
	if tc.Info.LastTokenUsage.TotalTokens != 1200 {
		t.Errorf("LastTokenUsage.TotalTokens = %d", tc.Info.LastTokenUsage.TotalTokens)
	}

	// model_context_window
	if tc.Info.ModelContextWindow != 128000 {
		t.Errorf("ModelContextWindow = %d", tc.Info.ModelContextWindow)
	}

	// rate_limits
	if tc.RateLimits == nil {
		t.Fatal("RateLimits is nil, expected non-nil")
	}
	if tc.RateLimits.Primary.UsedPercent != 42.5 {
		t.Errorf("Primary.UsedPercent = %f", tc.RateLimits.Primary.UsedPercent)
	}
	if tc.RateLimits.Primary.WindowMinutes != 1 {
		t.Errorf("Primary.WindowMinutes = %d", tc.RateLimits.Primary.WindowMinutes)
	}
	if tc.RateLimits.Primary.ResetsAt != 1717322460 {
		t.Errorf("Primary.ResetsAt = %d", tc.RateLimits.Primary.ResetsAt)
	}
	if tc.RateLimits.Secondary.UsedPercent != 10.0 {
		t.Errorf("Secondary.UsedPercent = %f", tc.RateLimits.Secondary.UsedPercent)
	}
	if tc.RateLimits.Secondary.WindowMinutes != 60 {
		t.Errorf("Secondary.WindowMinutes = %d", tc.RateLimits.Secondary.WindowMinutes)
	}
	if tc.RateLimits.Secondary.ResetsAt != 1717325400 {
		t.Errorf("Secondary.ResetsAt = %d", tc.RateLimits.Secondary.ResetsAt)
	}
}

func TestAsTokenCount_NoRateLimits(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleTokenCountNoRateLimits), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	tc, err := ev.AsTokenCount()
	if err != nil {
		t.Fatalf("AsTokenCount: %v", err)
	}

	if tc.RateLimits != nil {
		t.Errorf("RateLimits = %+v, want nil", tc.RateLimits)
	}
	if tc.Info.TotalTokenUsage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", tc.Info.TotalTokenUsage.TotalTokens)
	}
}

// ---------------------------------------------------------------------------
// AsFunctionCall
// ---------------------------------------------------------------------------

func TestAsFunctionCall(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(sampleFunctionCall), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	fc, err := ev.AsFunctionCall()
	if err != nil {
		t.Fatalf("AsFunctionCall: %v", err)
	}

	if fc.Name != "shell" {
		t.Errorf("Name = %q", fc.Name)
	}
	if fc.Arguments != `{"cmd":"ls -la"}` {
		t.Errorf("Arguments = %q", fc.Arguments)
	}
	if fc.CallID != "call_xyz789" {
		t.Errorf("CallID = %q", fc.CallID)
	}
}
