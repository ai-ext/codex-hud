package parser

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseLine
// ---------------------------------------------------------------------------

func TestParseLine_Valid(t *testing.T) {
	line := `{"timestamp":"2025-06-02T10:00:00.000Z","type":"session_meta","payload":{"id":"sess_abc123","timestamp":"2025-06-02T10:00:00.000Z","cwd":"/tmp","originator":"codex-cli","cli_version":"0.1.0","source":"terminal","model_provider":"openai"}}`
	ev, err := ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if ev.Type != "session_meta" {
		t.Errorf("Type = %q, want %q", ev.Type, "session_meta")
	}
	if ev.Timestamp != "2025-06-02T10:00:00.000Z" {
		t.Errorf("Timestamp = %q", ev.Timestamp)
	}
}

func TestParseLine_WithWhitespace(t *testing.T) {
	line := `   {"timestamp":"2025-06-02T10:00:00.000Z","type":"session_meta","payload":{}}   `
	ev, err := ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if ev.Type != "session_meta" {
		t.Errorf("Type = %q", ev.Type)
	}
}

func TestParseLine_EmptyLine(t *testing.T) {
	_, err := ParseLine("")
	if err == nil {
		t.Fatal("expected error for empty line")
	}
}

func TestParseLine_BlankLine(t *testing.T) {
	_, err := ParseLine("   \t  ")
	if err == nil {
		t.Fatal("expected error for blank line")
	}
}

func TestParseLine_InvalidJSON(t *testing.T) {
	_, err := ParseLine(`{not valid json}`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// ParseLines
// ---------------------------------------------------------------------------

func TestParseLines_MultipleEvents(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2025-06-02T10:00:00.000Z","type":"session_meta","payload":{"id":"sess_1"}}`,
		`{"timestamp":"2025-06-02T10:00:01.000Z","type":"turn_context","payload":{"turn_id":"t1"}}`,
		`{"timestamp":"2025-06-02T10:00:02.000Z","type":"response_item","payload":{"subtype":"function_call","name":"shell","arguments":"{}","call_id":"c1"}}`,
	}, "\n")

	events, errs := ParseLines(strings.NewReader(input))
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].Type != "session_meta" {
		t.Errorf("events[0].Type = %q", events[0].Type)
	}
	if events[1].Type != "turn_context" {
		t.Errorf("events[1].Type = %q", events[1].Type)
	}
	if events[2].Type != "response_item" {
		t.Errorf("events[2].Type = %q", events[2].Type)
	}
}

func TestParseLines_SkipsBlankLines(t *testing.T) {
	input := "\n\n" + `{"timestamp":"t","type":"session_meta","payload":{}}` + "\n\n"
	events, errs := ParseLines(strings.NewReader(input))
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
}

func TestParseLines_CollectsErrors(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"t","type":"session_meta","payload":{}}`,
		`{bad json}`,
		`{"timestamp":"t2","type":"turn_context","payload":{}}`,
	}, "\n")

	events, errs := ParseLines(strings.NewReader(input))
	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1: %v", len(errs), errs)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestParseLines_EmptyInput(t *testing.T) {
	events, errs := ParseLines(strings.NewReader(""))
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(events) != 0 {
		t.Errorf("got %d events, want 0", len(events))
	}
}

func TestParseLines_RealisticSession(t *testing.T) {
	// Simulates a short Codex session with multiple event types
	input := strings.Join([]string{
		`{"timestamp":"2025-06-02T10:00:00.000Z","type":"session_meta","payload":{"id":"sess_abc123","timestamp":"2025-06-02T10:00:00.000Z","cwd":"/home/user/project","originator":"codex-cli","cli_version":"0.1.2025060200","source":"terminal","model_provider":"openai"}}`,
		`{"timestamp":"2025-06-02T10:00:01.000Z","type":"turn_context","payload":{"turn_id":"turn_001","cwd":"/home/user/project","model":"o4-mini","personality":"You are helpful.","approval_policy":"auto-edit","sandbox_policy":{"type":"docker"},"collaboration_mode":{"mode":"pair","settings":{"model":"o4-mini","reasoning_effort":"medium"}}}}`,
		`{"timestamp":"2025-06-02T10:00:03.000Z","type":"response_item","payload":{"subtype":"function_call","name":"shell","arguments":"{\"cmd\":\"ls -la\"}","call_id":"call_xyz789"}}`,
		`{"timestamp":"2025-06-02T10:00:05.000Z","type":"event_msg","payload":{"subtype":"token_count","info":{"total_token_usage":{"input_tokens":5000,"cached_input_tokens":1200,"output_tokens":800,"reasoning_output_tokens":200,"total_tokens":6000},"last_token_usage":{"input_tokens":1000,"cached_input_tokens":300,"output_tokens":150,"reasoning_output_tokens":50,"total_tokens":1200},"model_context_window":128000},"rate_limits":{"primary":{"used_percent":42.5,"window_minutes":1,"resets_at":1717322460},"secondary":{"used_percent":10.0,"window_minutes":60,"resets_at":1717325400}}}}`,
	}, "\n")

	events, errs := ParseLines(strings.NewReader(input))
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}

	// Verify we can parse each typed payload
	meta, err := events[0].AsSessionMeta()
	if err != nil {
		t.Fatalf("AsSessionMeta: %v", err)
	}
	if meta.ID != "sess_abc123" {
		t.Errorf("meta.ID = %q", meta.ID)
	}

	tc, err := events[1].AsTurnContext()
	if err != nil {
		t.Fatalf("AsTurnContext: %v", err)
	}
	if tc.TurnID != "turn_001" {
		t.Errorf("tc.TurnID = %q", tc.TurnID)
	}

	fc, err := events[2].AsFunctionCall()
	if err != nil {
		t.Fatalf("AsFunctionCall: %v", err)
	}
	if fc.Name != "shell" {
		t.Errorf("fc.Name = %q", fc.Name)
	}

	tok, err := events[3].AsTokenCount()
	if err != nil {
		t.Fatalf("AsTokenCount: %v", err)
	}
	if tok.Info.TotalTokenUsage.InputTokens != 5000 {
		t.Errorf("tok InputTokens = %d", tok.Info.TotalTokenUsage.InputTokens)
	}
}
