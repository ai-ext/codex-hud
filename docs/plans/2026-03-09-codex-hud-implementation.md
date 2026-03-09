# codex-hud Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Go + bubbletea로 크로스 플랫폼 Codex CLI HUD를 만든다. `~/.codex/sessions/*.jsonl` 실시간 파싱, 카드 기반 TUI, wrapper 모드(codex + HUD 동시 실행), 환경 자동 감지.

**Architecture:** Passive File Watcher 방식. fsnotify로 `.jsonl` 감시 → 파서가 구조체로 변환 → State Store에 집약 → bubbletea TUI 렌더링. Wrapper 모드는 환경(tmux/WT/fallback)을 감지하여 codex 프로세스와 HUD를 분할 패널로 동시 기동.

**Tech Stack:** Go 1.22+, bubbletea, lipgloss, bubbles, fsnotify, cobra, BurntSushi/toml

**Design Doc:** `docs/plans/2026-03-07-codex-hud-design.md`

**Test Data:** `~/.codex/sessions/2026/03/07/rollout-2026-03-07T15-34-58-019cc701-9ee7-7f70-89c3-ff74e9041127.jsonl` (실제 Codex 세션 로그)

---

### Task 0: Go 설치 + 프로젝트 초기화

**Files:**
- Create: `go.mod`
- Create: `cmd/codex-hud/main.go`
- Create: `Makefile`

**Step 1: Go 설치**

Run: `brew install go`
Expected: Go 1.22+ 설치 완료

**Step 2: Go 모듈 초기화**

Run:
```bash
cd /Users/ds/codex-hud
go mod init github.com/ds/codex-hud
```
Expected: `go.mod` 생성

**Step 3: 최소 main.go 작성**

```go
// cmd/codex-hud/main.go
package main

import "fmt"

func main() {
	fmt.Println("codex-hud v0.1.0")
}
```

**Step 4: 빌드 + 실행 확인**

Run: `go run ./cmd/codex-hud`
Expected: `codex-hud v0.1.0` 출력

**Step 5: Makefile 작성**

```makefile
.PHONY: build run test clean build-all

build:
	go build -o dist/codex-hud ./cmd/codex-hud

run:
	go run ./cmd/codex-hud

test:
	go test ./... -v

clean:
	rm -rf dist/

build-all:
	GOOS=darwin  GOARCH=amd64 go build -o dist/codex-hud-darwin-amd64 ./cmd/codex-hud
	GOOS=darwin  GOARCH=arm64 go build -o dist/codex-hud-darwin-arm64 ./cmd/codex-hud
	GOOS=linux   GOARCH=amd64 go build -o dist/codex-hud-linux-amd64 ./cmd/codex-hud
	GOOS=windows GOARCH=amd64 go build -o dist/codex-hud-windows-amd64.exe ./cmd/codex-hud
```

**Step 6: Git 초기화 + 첫 커밋**

```bash
cd /Users/ds/codex-hud
git init
cat > .gitignore << 'EOF'
dist/
*.exe
.DS_Store
EOF
git add .
git commit -m "chore: initial project setup with Go module and Makefile"
```

---

### Task 1: JSONL Parser — 타입 정의

**Files:**
- Create: `internal/parser/types.go`
- Create: `internal/parser/types_test.go`

**Step 1: 테스트 작성**

```go
// internal/parser/types_test.go
package parser

import (
	"encoding/json"
	"testing"
)

func TestParseSessionMeta(t *testing.T) {
	raw := `{"timestamp":"2026-03-07T06:35:24.335Z","type":"session_meta","payload":{"id":"019cc701-9ee7-7f70-89c3-ff74e9041127","timestamp":"2026-03-07T06:34:58.180Z","cwd":"/Users/ds","originator":"codex_cli_rs","cli_version":"0.111.0","source":"cli","model_provider":"openai"}}`

	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if event.Type != "session_meta" {
		t.Errorf("expected session_meta, got %s", event.Type)
	}

	meta, err := event.AsSessionMeta()
	if err != nil {
		t.Fatalf("AsSessionMeta error: %v", err)
	}
	if meta.ID != "019cc701-9ee7-7f70-89c3-ff74e9041127" {
		t.Errorf("unexpected ID: %s", meta.ID)
	}
	if meta.CLIVersion != "0.111.0" {
		t.Errorf("unexpected CLI version: %s", meta.CLIVersion)
	}
}

func TestParseTurnContext(t *testing.T) {
	raw := `{"type":"turn_context","payload":{"turn_id":"abc","cwd":"/Users/ds","model":"gpt-5.4","personality":"pragmatic","collaboration_mode":{"mode":"default","settings":{"model":"gpt-5.4","reasoning_effort":"medium"}},"approval_policy":"untrusted","sandbox_policy":{"type":"workspace-write"}}}`

	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tc, err := event.AsTurnContext()
	if err != nil {
		t.Fatalf("AsTurnContext error: %v", err)
	}
	if tc.Model != "gpt-5.4" {
		t.Errorf("unexpected model: %s", tc.Model)
	}
	if tc.CollaborationMode.Settings.ReasoningEffort != "medium" {
		t.Errorf("unexpected reasoning effort: %s", tc.CollaborationMode.Settings.ReasoningEffort)
	}
}

func TestParseTokenCount(t *testing.T) {
	raw := `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1385034,"cached_input_tokens":1270784,"output_tokens":11636,"reasoning_output_tokens":2548,"total_tokens":1396670},"last_token_usage":{"input_tokens":70285,"cached_input_tokens":64384,"output_tokens":615,"reasoning_output_tokens":99,"total_tokens":70900},"model_context_window":258400},"rate_limits":null}}`

	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tc, err := event.AsTokenCount()
	if err != nil {
		t.Fatalf("AsTokenCount error: %v", err)
	}
	if tc.Info.TotalTokenUsage.InputTokens != 1385034 {
		t.Errorf("unexpected input tokens: %d", tc.Info.TotalTokenUsage.InputTokens)
	}
	if tc.Info.ModelContextWindow != 258400 {
		t.Errorf("unexpected context window: %d", tc.Info.ModelContextWindow)
	}
}

func TestParseTokenCountWithRateLimits(t *testing.T) {
	raw := `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":5,"total_tokens":120},"last_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":5,"total_tokens":120},"model_context_window":128000},"rate_limits":{"primary":{"used_percent":12.5,"window_minutes":300,"resets_at":1769251526},"secondary":{"used_percent":4.0,"window_minutes":10080,"resets_at":1769392822}}}}`

	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tc, err := event.AsTokenCount()
	if err != nil {
		t.Fatalf("AsTokenCount error: %v", err)
	}
	if tc.RateLimits == nil {
		t.Fatal("expected rate_limits to be non-nil")
	}
	if tc.RateLimits.Primary.UsedPercent != 12.5 {
		t.Errorf("unexpected primary used_percent: %f", tc.RateLimits.Primary.UsedPercent)
	}
}

func TestParseFunctionCall(t *testing.T) {
	raw := `{"type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"call_abc"}}`

	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	fc, err := event.AsFunctionCall()
	if err != nil {
		t.Fatalf("AsFunctionCall error: %v", err)
	}
	if fc.Name != "exec_command" {
		t.Errorf("unexpected name: %s", fc.Name)
	}
	if fc.CallID != "call_abc" {
		t.Errorf("unexpected call_id: %s", fc.CallID)
	}
}
```

**Step 2: 테스트 실행해서 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/parser/ -v`
Expected: FAIL — types not defined

**Step 3: 타입 구현**

```go
// internal/parser/types.go
package parser

import (
	"encoding/json"
	"fmt"
)

// Event is the top-level envelope for every JSONL line.
type Event struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// --- session_meta ---

type SessionMeta struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	Originator    string `json:"originator"`
	CLIVersion    string `json:"cli_version"`
	Source        string `json:"source"`
	ModelProvider string `json:"model_provider"`
}

func (e *Event) AsSessionMeta() (*SessionMeta, error) {
	if e.Type != "session_meta" {
		return nil, fmt.Errorf("event type is %q, not session_meta", e.Type)
	}
	var m SessionMeta
	if err := json.Unmarshal(e.Payload, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// --- turn_context ---

type TurnContext struct {
	TurnID            string            `json:"turn_id"`
	CWD               string            `json:"cwd"`
	Model             string            `json:"model"`
	Personality       string            `json:"personality"`
	ApprovalPolicy    string            `json:"approval_policy"`
	SandboxPolicy     SandboxPolicy     `json:"sandbox_policy"`
	CollaborationMode CollaborationMode `json:"collaboration_mode"`
}

type SandboxPolicy struct {
	Type string `json:"type"`
}

type CollaborationMode struct {
	Mode     string              `json:"mode"`
	Settings CollaborationSettings `json:"settings"`
}

type CollaborationSettings struct {
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

func (e *Event) AsTurnContext() (*TurnContext, error) {
	if e.Type != "turn_context" {
		return nil, fmt.Errorf("event type is %q, not turn_context", e.Type)
	}
	var tc TurnContext
	if err := json.Unmarshal(e.Payload, &tc); err != nil {
		return nil, err
	}
	return &tc, nil
}

// --- event_msg ---

type EventMsg struct {
	Type string `json:"type"`
	// Subtype-specific fields are parsed lazily via RawPayload.
	RawPayload json.RawMessage `json:"-"`
}

// We need a custom unmarshal to capture the full payload for subtype parsing.
func (e *Event) eventMsgType() (string, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(e.Payload, &envelope); err != nil {
		return "", err
	}
	return envelope.Type, nil
}

// --- token_count ---

type TokenCount struct {
	Info       *TokenInfo  `json:"info"`
	RateLimits *RateLimits `json:"rate_limits"`
}

type TokenInfo struct {
	TotalTokenUsage  TokenUsage `json:"total_token_usage"`
	LastTokenUsage   TokenUsage `json:"last_token_usage"`
	ModelContextWindow int      `json:"model_context_window"`
}

type TokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

type RateLimits struct {
	Primary   RateLimit `json:"primary"`
	Secondary RateLimit `json:"secondary"`
}

type RateLimit struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int     `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"`
}

func (e *Event) AsTokenCount() (*TokenCount, error) {
	if e.Type != "event_msg" {
		return nil, fmt.Errorf("event type is %q, not event_msg", e.Type)
	}
	subtype, err := e.eventMsgType()
	if err != nil {
		return nil, err
	}
	if subtype != "token_count" {
		return nil, fmt.Errorf("event_msg subtype is %q, not token_count", subtype)
	}
	var tc TokenCount
	if err := json.Unmarshal(e.Payload, &tc); err != nil {
		return nil, err
	}
	return &tc, nil
}

// --- response_item: function_call ---

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
}

func (e *Event) AsFunctionCall() (*FunctionCall, error) {
	if e.Type != "response_item" {
		return nil, fmt.Errorf("event type is %q, not response_item", e.Type)
	}
	subtype, err := e.eventMsgType()
	if err != nil {
		return nil, err
	}
	if subtype != "function_call" {
		return nil, fmt.Errorf("response_item subtype is %q, not function_call", subtype)
	}
	var fc FunctionCall
	if err := json.Unmarshal(e.Payload, &fc); err != nil {
		return nil, err
	}
	return &fc, nil
}

// --- event_msg: task_started / task_complete ---

type TaskStarted struct {
	TurnID             string `json:"turn_id"`
	ModelContextWindow int    `json:"model_context_window"`
}

type TaskComplete struct {
	TurnID           string `json:"turn_id"`
	LastAgentMessage string `json:"last_agent_message"`
}
```

**Step 4: 테스트 실행해서 통과 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/parser/ -v`
Expected: PASS (5 tests)

**Step 5: 커밋**

```bash
git add internal/parser/
git commit -m "feat: add JSONL parser types with tests for session_meta, turn_context, token_count, function_call"
```

---

### Task 2: JSONL Parser — 라인 파서

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

**Step 1: 테스트 작성**

```go
// internal/parser/parser_test.go
package parser

import (
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	line := `{"timestamp":"2026-03-07T06:35:24.335Z","type":"session_meta","payload":{"id":"test-id","cli_version":"0.111.0","cwd":"/Users/ds","originator":"codex_cli_rs","source":"cli","model_provider":"openai"}}`

	event, err := ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine error: %v", err)
	}
	if event.Type != "session_meta" {
		t.Errorf("expected session_meta, got %s", event.Type)
	}
}

func TestParseLineInvalid(t *testing.T) {
	_, err := ParseLine("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseLineEmpty(t *testing.T) {
	_, err := ParseLine("")
	if err == nil {
		t.Error("expected error for empty line")
	}
}

func TestParseLines(t *testing.T) {
	input := strings.NewReader(`{"type":"session_meta","payload":{"id":"a","cli_version":"0.111.0","cwd":"/tmp","originator":"codex_cli_rs","source":"cli","model_provider":"openai"}}
{"type":"turn_context","payload":{"turn_id":"b","cwd":"/tmp","model":"gpt-5.4","personality":"pragmatic","collaboration_mode":{"mode":"default","settings":{"model":"gpt-5.4","reasoning_effort":"medium"}},"approval_policy":"untrusted","sandbox_policy":{"type":"workspace-write"}}}
not valid json
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":5,"total_tokens":120},"last_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20,"reasoning_output_tokens":5,"total_tokens":120},"model_context_window":128000},"rate_limits":null}}
`)

	events, errs := ParseLines(input)
	if len(events) != 3 {
		t.Errorf("expected 3 valid events, got %d", len(events))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
	if events[0].Type != "session_meta" {
		t.Errorf("first event should be session_meta, got %s", events[0].Type)
	}
}
```

**Step 2: 테스트 실행해서 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/parser/ -v -run TestParseLine`
Expected: FAIL — ParseLine not defined

**Step 3: 구현**

```go
// internal/parser/parser.go
package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParseLine parses a single JSONL line into an Event.
func ParseLine(line string) (*Event, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}
	var event Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, fmt.Errorf("json parse error: %w", err)
	}
	return &event, nil
}

// ParseLines reads all lines from a reader, parsing each as an Event.
// Returns successfully parsed events and any errors encountered.
func ParseLines(r io.Reader) ([]*Event, []error) {
	var events []*Event
	var errs []error

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large lines
	for scanner.Scan() {
		event, err := ParseLine(scanner.Text())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("scanner error: %w", err))
	}
	return events, errs
}
```

**Step 4: 테스트 통과 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/parser/ -v`
Expected: PASS (all tests)

**Step 5: 커밋**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: add JSONL line parser with ParseLine and ParseLines"
```

---

### Task 3: State Store

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/state/state_test.go`

**Step 1: 테스트 작성**

```go
// internal/state/state_test.go
package state

import (
	"testing"

	"github.com/ds/codex-hud/internal/parser"
)

func TestNewSession(t *testing.T) {
	s := New()
	if s.TurnCount != 0 {
		t.Errorf("expected 0 turns, got %d", s.TurnCount)
	}
	if s.Model != "" {
		t.Errorf("expected empty model, got %s", s.Model)
	}
}

func TestApplySessionMeta(t *testing.T) {
	s := New()
	s.ApplySessionMeta(&parser.SessionMeta{
		ID:            "test-id",
		CLIVersion:    "0.111.0",
		CWD:           "/Users/ds/project",
		ModelProvider:  "openai",
	})
	if s.SessionID != "test-id" {
		t.Errorf("expected test-id, got %s", s.SessionID)
	}
	if s.CLIVersion != "0.111.0" {
		t.Errorf("expected 0.111.0, got %s", s.CLIVersion)
	}
	if s.CWD != "/Users/ds/project" {
		t.Errorf("expected /Users/ds/project, got %s", s.CWD)
	}
}

func TestApplyTurnContext(t *testing.T) {
	s := New()
	s.ApplyTurnContext(&parser.TurnContext{
		Model:          "gpt-5.4",
		ApprovalPolicy: "untrusted",
		SandboxPolicy:  parser.SandboxPolicy{Type: "workspace-write"},
		CollaborationMode: parser.CollaborationMode{
			Settings: parser.CollaborationSettings{ReasoningEffort: "medium"},
		},
	})
	if s.Model != "gpt-5.4" {
		t.Errorf("expected gpt-5.4, got %s", s.Model)
	}
	if s.ReasoningEffort != "medium" {
		t.Errorf("expected medium, got %s", s.ReasoningEffort)
	}
	if s.ApprovalPolicy != "untrusted" {
		t.Errorf("expected untrusted, got %s", s.ApprovalPolicy)
	}
}

func TestApplyTokenCount(t *testing.T) {
	s := New()
	s.ApplyTokenCount(&parser.TokenCount{
		Info: &parser.TokenInfo{
			TotalTokenUsage: parser.TokenUsage{
				InputTokens:       1385034,
				CachedInputTokens: 1270784,
				OutputTokens:      11636,
				ReasoningOutputTokens: 2548,
			},
			LastTokenUsage: parser.TokenUsage{
				InputTokens: 70285,
			},
			ModelContextWindow: 258400,
		},
	})
	if s.TotalInputTokens != 1385034 {
		t.Errorf("unexpected total input: %d", s.TotalInputTokens)
	}
	if s.ContextWindowSize != 258400 {
		t.Errorf("unexpected context window: %d", s.ContextWindowSize)
	}
	if s.ContextUsedTokens != 70285 {
		t.Errorf("unexpected context used: %d", s.ContextUsedTokens)
	}

	pct := s.ContextPercent()
	if pct < 27.0 || pct > 28.0 {
		t.Errorf("expected ~27%%, got %.1f%%", pct)
	}
}

func TestApplyTokenCountWithRateLimits(t *testing.T) {
	s := New()
	s.ApplyTokenCount(&parser.TokenCount{
		Info: &parser.TokenInfo{
			ModelContextWindow: 128000,
			LastTokenUsage:     parser.TokenUsage{InputTokens: 64000},
		},
		RateLimits: &parser.RateLimits{
			Primary:   parser.RateLimit{UsedPercent: 12.5, ResetsAt: 1769251526},
			Secondary: parser.RateLimit{UsedPercent: 4.0, ResetsAt: 1769392822},
		},
	})
	if !s.HasRateLimits {
		t.Error("expected HasRateLimits = true")
	}
	if s.PrimaryRatePercent != 12.5 {
		t.Errorf("unexpected primary rate: %f", s.PrimaryRatePercent)
	}
}

func TestTrackToolCalls(t *testing.T) {
	s := New()
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_1",
	})
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_2",
	})
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "apply_patch",
		CallID: "call_3",
	})

	if s.ToolCounts["exec_command"] != 2 {
		t.Errorf("expected 2 exec_command, got %d", s.ToolCounts["exec_command"])
	}
	if s.ToolCounts["apply_patch"] != 1 {
		t.Errorf("expected 1 apply_patch, got %d", s.ToolCounts["apply_patch"])
	}
	if len(s.ActiveTools) != 3 {
		t.Errorf("expected 3 active tools, got %d", len(s.ActiveTools))
	}
}

func TestCompleteToolCall(t *testing.T) {
	s := New()
	s.ApplyFunctionCall(&parser.FunctionCall{
		Name:   "exec_command",
		CallID: "call_1",
	})
	if len(s.ActiveTools) != 1 {
		t.Fatalf("expected 1 active tool, got %d", len(s.ActiveTools))
	}

	s.CompleteFunctionCall("call_1")
	if len(s.ActiveTools) != 0 {
		t.Errorf("expected 0 active tools after completion, got %d", len(s.ActiveTools))
	}
}

func TestIncrementTurns(t *testing.T) {
	s := New()
	s.IncrementTurn()
	s.IncrementTurn()
	if s.TurnCount != 2 {
		t.Errorf("expected 2 turns, got %d", s.TurnCount)
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/state/ -v`
Expected: FAIL — package not found

**Step 3: 구현**

```go
// internal/state/state.go
package state

import (
	"sync"
	"time"

	"github.com/ds/codex-hud/internal/parser"
)

// ActiveTool tracks a currently running tool call.
type ActiveTool struct {
	Name    string
	CallID  string
	StartAt time.Time
}

// Session holds the aggregated state of a Codex session.
type Session struct {
	mu sync.RWMutex

	// Session info
	SessionID     string
	CLIVersion    string
	CWD           string
	ModelProvider string
	StartTime     time.Time

	// Model info
	Model           string
	ReasoningEffort string
	ApprovalPolicy  string
	SandboxType     string

	// Context window
	ContextWindowSize int
	ContextUsedTokens int

	// Token totals
	TotalInputTokens    int
	TotalCachedTokens   int
	TotalOutputTokens   int
	TotalReasonTokens   int

	// Rate limits
	HasRateLimits       bool
	PrimaryRatePercent  float64
	PrimaryResetsAt     int64
	SecondaryRatePercent float64
	SecondaryResetsAt   int64

	// Turn tracking
	TurnCount int

	// Tool tracking
	ToolCounts  map[string]int
	ActiveTools []ActiveTool
}

// New creates a new empty Session state.
func New() *Session {
	return &Session{
		StartTime:  time.Now(),
		ToolCounts: make(map[string]int),
	}
}

func (s *Session) ApplySessionMeta(m *parser.SessionMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SessionID = m.ID
	s.CLIVersion = m.CLIVersion
	s.CWD = m.CWD
	s.ModelProvider = m.ModelProvider
	if m.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, m.Timestamp); err == nil {
			s.StartTime = t
		}
	}
}

func (s *Session) ApplyTurnContext(tc *parser.TurnContext) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Model = tc.Model
	s.ApprovalPolicy = tc.ApprovalPolicy
	s.SandboxType = tc.SandboxPolicy.Type
	s.ReasoningEffort = tc.CollaborationMode.Settings.ReasoningEffort
	if tc.CWD != "" {
		s.CWD = tc.CWD
	}
}

func (s *Session) ApplyTokenCount(tc *parser.TokenCount) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tc.Info != nil {
		s.TotalInputTokens = tc.Info.TotalTokenUsage.InputTokens
		s.TotalCachedTokens = tc.Info.TotalTokenUsage.CachedInputTokens
		s.TotalOutputTokens = tc.Info.TotalTokenUsage.OutputTokens
		s.TotalReasonTokens = tc.Info.TotalTokenUsage.ReasoningOutputTokens
		s.ContextWindowSize = tc.Info.ModelContextWindow
		s.ContextUsedTokens = tc.Info.LastTokenUsage.InputTokens
	}

	if tc.RateLimits != nil {
		s.HasRateLimits = true
		s.PrimaryRatePercent = tc.RateLimits.Primary.UsedPercent
		s.PrimaryResetsAt = tc.RateLimits.Primary.ResetsAt
		s.SecondaryRatePercent = tc.RateLimits.Secondary.UsedPercent
		s.SecondaryResetsAt = tc.RateLimits.Secondary.ResetsAt
	}
}

// ContextPercent returns the context window usage as a percentage.
func (s *Session) ContextPercent() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ContextWindowSize == 0 {
		return 0
	}
	return float64(s.ContextUsedTokens) / float64(s.ContextWindowSize) * 100
}

func (s *Session) ApplyFunctionCall(fc *parser.FunctionCall) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ToolCounts[fc.Name]++
	s.ActiveTools = append(s.ActiveTools, ActiveTool{
		Name:    fc.Name,
		CallID:  fc.CallID,
		StartAt: time.Now(),
	})
}

func (s *Session) CompleteFunctionCall(callID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, tool := range s.ActiveTools {
		if tool.CallID == callID {
			s.ActiveTools = append(s.ActiveTools[:i], s.ActiveTools[i+1:]...)
			return
		}
	}
}

func (s *Session) IncrementTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TurnCount++
}
```

**Step 4: 테스트 통과 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/state/ -v`
Expected: PASS

**Step 5: 커밋**

```bash
git add internal/state/
git commit -m "feat: add session state store with token, tool, and turn tracking"
```

---

### Task 4: File Watcher

**Files:**
- Create: `internal/watcher/watcher.go`
- Create: `internal/watcher/watcher_test.go`

**Step 1: 테스트 작성**

```go
// internal/watcher/watcher_test.go
package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindLatestSession(t *testing.T) {
	// Create temp dir structure mimicking ~/.codex/sessions/
	tmpDir := t.TempDir()
	dayDir := filepath.Join(tmpDir, "2026", "03", "07")
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create two session files with different timestamps
	file1 := filepath.Join(dayDir, "rollout-2026-03-07T10-00-00-aaa.jsonl")
	file2 := filepath.Join(dayDir, "rollout-2026-03-07T15-00-00-bbb.jsonl")

	os.WriteFile(file1, []byte(`{"type":"session_meta"}`+"\n"), 0644)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(file2, []byte(`{"type":"session_meta"}`+"\n"), 0644)

	latest, err := FindLatestSession(tmpDir)
	if err != nil {
		t.Fatalf("FindLatestSession error: %v", err)
	}
	if latest != file2 {
		t.Errorf("expected %s, got %s", file2, latest)
	}
}

func TestFindLatestSessionEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := FindLatestSession(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestTailFile(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(fpath, []byte("line1\nline2\n"), 0644)

	lines := make(chan string, 10)
	errCh := make(chan error, 1)

	go func() {
		errCh <- TailFile(fpath, lines, make(chan struct{}))
	}()

	// Should read existing lines
	select {
	case l := <-lines:
		if l != "line1" {
			t.Errorf("expected line1, got %s", l)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for line1")
	}

	select {
	case l := <-lines:
		if l != "line2" {
			t.Errorf("expected line2, got %s", l)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for line2")
	}

	// Append a new line
	f, _ := os.OpenFile(fpath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("line3\n")
	f.Close()

	select {
	case l := <-lines:
		if l != "line3" {
			t.Errorf("expected line3, got %s", l)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for appended line3")
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/watcher/ -v -timeout 30s`
Expected: FAIL — functions not defined

**Step 3: 구현**

```go
// internal/watcher/watcher.go
package watcher

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// FindLatestSession finds the most recently modified .jsonl file under sessionsDir.
func FindLatestSession(sessionsDir string) (string, error) {
	var files []string

	err := filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking sessions dir: %w", err)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no .jsonl files found in %s", sessionsDir)
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		if infoI == nil || infoJ == nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files[0], nil
}

// TailFile reads existing content from a file and then watches for new appended lines.
// Lines are sent to the lines channel. Blocks until stop is closed or an error occurs.
func TailFile(path string, lines chan<- string, stop <-chan struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// Read existing content
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\n\r")
		if line != "" {
			select {
			case lines <- line:
			case <-stop:
				return nil
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
	}

	// Watch for new content
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		return fmt.Errorf("watching file: %w", err)
	}

	for {
		select {
		case <-stop:
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write != 0 {
				for {
					line, err := reader.ReadString('\n')
					line = strings.TrimRight(line, "\n\r")
					if line != "" {
						select {
						case lines <- line:
						case <-stop:
							return nil
						}
					}
					if err == io.EOF {
						break
					}
					if err != nil {
						return fmt.Errorf("reading appended content: %w", err)
					}
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
```

**Step 4: 의존성 설치 + 테스트 통과 확인**

Run:
```bash
cd /Users/ds/codex-hud
go get github.com/fsnotify/fsnotify
go test ./internal/watcher/ -v -timeout 30s
```
Expected: PASS

**Step 5: 커밋**

```bash
git add internal/watcher/ go.mod go.sum
git commit -m "feat: add file watcher with FindLatestSession and TailFile"
```

---

### Task 5: Git Status

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

**Step 1: 테스트 작성**

```go
// internal/git/git_test.go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup failed: %s: %v", string(out), err)
		}
	}

	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.Run()

	return dir
}

func TestGetStatusClean(t *testing.T) {
	dir := setupGitRepo(t)
	status, err := GetStatus(dir)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if status.Branch == "" {
		t.Error("expected non-empty branch")
	}
	if status.Dirty {
		t.Error("expected clean repo")
	}
}

func TestGetStatusDirty(t *testing.T) {
	dir := setupGitRepo(t)
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)

	status, err := GetStatus(dir)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if !status.Dirty {
		t.Error("expected dirty repo")
	}
	if status.Untracked != 1 {
		t.Errorf("expected 1 untracked, got %d", status.Untracked)
	}
}

func TestGetStatusNotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := GetStatus(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/git/ -v`
Expected: FAIL

**Step 3: 구현**

```go
// internal/git/git.go
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Status represents the git status of a directory.
type Status struct {
	Branch    string
	Dirty     bool
	Modified  int
	Added     int
	Deleted   int
	Untracked int
	Ahead     int
	Behind    int
}

const timeout = 1 * time.Second

// GetStatus returns the git status for the given directory.
func GetStatus(dir string) (*Status, error) {
	branch, err := getBranch(dir)
	if err != nil {
		return nil, fmt.Errorf("not a git repo or git not available: %w", err)
	}

	status := &Status{Branch: branch}

	porcelain, err := runGit(dir, "status", "--porcelain")
	if err != nil {
		return status, nil // return what we have
	}

	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 2 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "??"):
			status.Untracked++
		case line[0] == 'M' || line[1] == 'M':
			status.Modified++
		case line[0] == 'A':
			status.Added++
		case line[0] == 'D' || line[1] == 'D':
			status.Deleted++
		}
	}

	status.Dirty = status.Modified > 0 || status.Added > 0 || status.Deleted > 0 || status.Untracked > 0

	return status, nil
}

func getBranch(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func runGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

**Step 4: 테스트 통과 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/git/ -v`
Expected: PASS

**Step 5: 커밋**

```bash
git add internal/git/
git commit -m "feat: add git status reader with branch, dirty, and file stats"
```

---

### Task 6: Config

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: 테스트 작성**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Display.Theme != "default" {
		t.Errorf("expected default theme, got %s", cfg.Display.Theme)
	}
	if cfg.Display.RefreshMs != 500 {
		t.Errorf("expected 500ms refresh, got %d", cfg.Display.RefreshMs)
	}
	if !cfg.Display.ShowGit {
		t.Error("expected ShowGit to be true by default")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/codex-hud.toml")
	if err != nil {
		t.Fatalf("missing config should not error: %v", err)
	}
	if cfg.Display.Theme != "default" {
		t.Error("expected default config when file missing")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codex-hud.toml")

	content := `
[display]
theme = "neon"
refresh_ms = 250
show_rate_limit = false

[git]
show_ahead_behind = true
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Display.Theme != "neon" {
		t.Errorf("expected neon, got %s", cfg.Display.Theme)
	}
	if cfg.Display.RefreshMs != 250 {
		t.Errorf("expected 250, got %d", cfg.Display.RefreshMs)
	}
	if cfg.Display.ShowRateLimit {
		t.Error("expected ShowRateLimit to be false")
	}
	// Defaults should still apply for unset fields
	if !cfg.Display.ShowGit {
		t.Error("expected ShowGit to remain true (default)")
	}
	if !cfg.Git.ShowAheadBehind {
		t.Error("expected ShowAheadBehind to be true")
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codex-hud.toml")
	os.WriteFile(path, []byte("not valid toml [[["), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("invalid TOML should fall back to defaults: %v", err)
	}
	if cfg.Display.Theme != "default" {
		t.Error("expected default theme on invalid TOML")
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `cd /Users/ds/codex-hud && go test ./internal/config/ -v`
Expected: FAIL

**Step 3: 구현**

```go
// internal/config/config.go
package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Display DisplayConfig `toml:"display"`
	Git     GitConfig     `toml:"git"`
	Tmux    TmuxConfig    `toml:"tmux"`
}

type DisplayConfig struct {
	Theme         string `toml:"theme"`
	RefreshMs     int    `toml:"refresh_ms"`
	ShowRateLimit bool   `toml:"show_rate_limit"`
	ShowActivity  bool   `toml:"show_activity"`
	ShowGit       bool   `toml:"show_git"`
}

type GitConfig struct {
	ShowDirty       bool `toml:"show_dirty"`
	ShowAheadBehind bool `toml:"show_ahead_behind"`
	ShowFileStats   bool `toml:"show_file_stats"`
}

type TmuxConfig struct {
	AutoDetect bool   `toml:"auto_detect"`
	Position   string `toml:"position"`
	Size       int    `toml:"size"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Display: DisplayConfig{
			Theme:         "default",
			RefreshMs:     500,
			ShowRateLimit: true,
			ShowActivity:  true,
			ShowGit:       true,
		},
		Git: GitConfig{
			ShowDirty:       true,
			ShowAheadBehind: false,
			ShowFileStats:   false,
		},
		Tmux: TmuxConfig{
			AutoDetect: true,
			Position:   "bottom",
			Size:       30,
		},
	}
}

// Load reads a config from the given TOML file path.
// If the file doesn't exist or is invalid, returns defaults.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil // file missing → defaults
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return Default(), nil // invalid TOML → defaults
	}

	return cfg, nil
}
```

**Step 4: 의존성 + 테스트 통과 확인**

Run:
```bash
cd /Users/ds/codex-hud
go get github.com/BurntSushi/toml
go test ./internal/config/ -v
```
Expected: PASS

**Step 5: 커밋**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add TOML config with defaults and file loading"
```

---

### Task 7: TUI — Styles + Components

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/components/header.go`
- Create: `internal/tui/components/context.go`
- Create: `internal/tui/components/tokens.go`
- Create: `internal/tui/components/session.go`
- Create: `internal/tui/components/ratelimit.go`
- Create: `internal/tui/components/activity.go`

**Step 1: styles.go 작성**

```go
// internal/tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorCyan    = lipgloss.Color("86")
	ColorGreen   = lipgloss.Color("78")
	ColorYellow  = lipgloss.Color("220")
	ColorRed     = lipgloss.Color("196")
	ColorDim     = lipgloss.Color("240")
	ColorWhite   = lipgloss.Color("255")
	ColorBorder  = lipgloss.Color("240")

	// Base styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan)

	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	OuterStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorCyan).
			Padding(1, 2)
)

// ProgressBar renders a progress bar with the given width and percentage.
func ProgressBar(width int, percent float64, color lipgloss.Color) string {
	if width < 4 {
		width = 4
	}
	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	bar := lipgloss.NewStyle().Foreground(color).Render(repeatChar('█', filled))
	bar += lipgloss.NewStyle().Foreground(ColorDim).Render(repeatChar('░', empty))
	return bar
}

// BarColor returns the appropriate color for a percentage value.
func BarColor(percent float64, thresholds [2]float64) lipgloss.Color {
	if percent >= thresholds[1] {
		return ColorRed
	}
	if percent >= thresholds[0] {
		return ColorYellow
	}
	return ColorGreen
}

func repeatChar(ch rune, n int) string {
	result := make([]rune, n)
	for i := range result {
		result[i] = ch
	}
	return string(result)
}
```

**Step 2: 각 컴포넌트 작성**

각 컴포넌트는 `state.Session`을 받아서 string을 반환하는 `Render(s *state.Session, width int) string` 함수를 가짐.

```go
// internal/tui/components/header.go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderHeader(s *state.Session, width int) string {
	model := tui.TitleStyle.Render(fmt.Sprintf("● %s", s.Model))
	effort := tui.LabelStyle.Render(s.ReasoningEffort)
	policy := tui.LabelStyle.Render(s.ApprovalPolicy)
	version := tui.LabelStyle.Render(fmt.Sprintf("v%s", s.CLIVersion))

	return lipgloss.JoinHorizontal(lipgloss.Center,
		model, "    ", effort, "    ", policy, "    ", version,
	)
}
```

```go
// internal/tui/components/context.go
package components

import (
	"fmt"

	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderContext(s *state.Session, width int) string {
	pct := s.ContextPercent()
	barWidth := width - 12
	if barWidth < 10 {
		barWidth = 10
	}
	color := tui.BarColor(pct, [2]float64{50, 75})

	bar := tui.ProgressBar(barWidth, pct, color)
	pctStr := tui.ValueStyle.Render(fmt.Sprintf("%.1f%%", pct))
	detail := tui.LabelStyle.Render(fmt.Sprintf("%s / %s tokens",
		formatNumber(s.ContextUsedTokens), formatNumber(s.ContextWindowSize)))

	content := fmt.Sprintf("%s  %s\n%s", bar, pctStr, detail)
	return tui.CardStyle.Width(width).Render(
		fmt.Sprintf("%s\n%s", tui.TitleStyle.Render("Context"), content),
	)
}

func formatNumber(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%sK", fmt.Sprintf("%.1f", float64(n)/1_000))
	}
	return fmt.Sprintf("%d", n)
}
```

```go
// internal/tui/components/tokens.go
package components

import (
	"fmt"

	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderTokens(s *state.Session, width int) string {
	lines := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s",
		tui.LabelStyle.Render("↓ in    "), tui.ValueStyle.Render(formatNumber(s.TotalInputTokens)),
		tui.LabelStyle.Render("↻ cache "), tui.ValueStyle.Render(formatNumber(s.TotalCachedTokens)),
		tui.LabelStyle.Render("↑ out   "), tui.ValueStyle.Render(formatNumber(s.TotalOutputTokens)),
		tui.LabelStyle.Render("◆ reason"), tui.ValueStyle.Render(formatNumber(s.TotalReasonTokens)),
	)
	return tui.CardStyle.Width(width).Render(
		fmt.Sprintf("%s\n%s", tui.TitleStyle.Render("Tokens"), lines),
	)
}
```

```go
// internal/tui/components/session.go
package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderSession(s *state.Session, gitStatus *git.Status, width int) string {
	duration := time.Since(s.StartTime).Truncate(time.Second)
	durationStr := formatDuration(duration)
	turnsStr := fmt.Sprintf("%d turns", s.TurnCount)

	cwd := shortenPath(s.CWD)

	lines := fmt.Sprintf("%s %s   %s\n%s %s",
		tui.LabelStyle.Render("⏱"), tui.ValueStyle.Render(durationStr), tui.LabelStyle.Render(turnsStr),
		tui.LabelStyle.Render("📂"), tui.ValueStyle.Render(cwd),
	)

	if gitStatus != nil {
		branchStr := gitStatus.Branch
		if gitStatus.Dirty {
			branchStr += "*"
		}
		gitLine := fmt.Sprintf("%s %s", tui.LabelStyle.Render("🌿"), tui.ValueStyle.Render(branchStr))
		if gitStatus.Modified > 0 || gitStatus.Added > 0 || gitStatus.Deleted > 0 {
			gitLine += tui.LabelStyle.Render(fmt.Sprintf("  +%d ~%d -%d",
				gitStatus.Added, gitStatus.Modified, gitStatus.Deleted))
		}
		lines += "\n" + gitLine
	}

	return tui.CardStyle.Width(width).Render(
		fmt.Sprintf("%s\n%s", tui.TitleStyle.Render("Session"), lines),
	)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

func shortenPath(p string) string {
	home, _ := filepath.Abs(filepath.Join("~"))
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	// Try $HOME
	if h := filepath.Dir(filepath.Dir(p)); strings.HasSuffix(h, "/Users") || strings.HasSuffix(h, "\\Users") {
		parts := strings.SplitN(p, string(filepath.Separator), 4)
		if len(parts) >= 4 {
			return "~/" + parts[3]
		}
	}
	return p
}
```

```go
// internal/tui/components/ratelimit.go
package components

import (
	"fmt"
	"time"

	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderRateLimit(s *state.Session, width int) string {
	if !s.HasRateLimits {
		return ""
	}

	pct := s.PrimaryRatePercent
	barWidth := width - 12
	if barWidth < 10 {
		barWidth = 10
	}
	color := tui.BarColor(pct, [2]float64{50, 80})

	bar := tui.ProgressBar(barWidth, pct, color)
	pctStr := tui.ValueStyle.Render(fmt.Sprintf("%.0f%%", pct))

	resetStr := ""
	if s.PrimaryResetsAt > 0 {
		remaining := time.Until(time.Unix(s.PrimaryResetsAt, 0))
		if remaining > 0 {
			resetStr = tui.LabelStyle.Render(fmt.Sprintf("resets in %s", formatDuration(remaining.Truncate(time.Second))))
		}
	}

	content := fmt.Sprintf("%s  %s\n%s", bar, pctStr, resetStr)
	return tui.CardStyle.Width(width).Render(
		fmt.Sprintf("%s\n%s", tui.TitleStyle.Render("Rate Limit"), content),
	)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
```

```go
// internal/tui/components/activity.go
package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
	"github.com/ds/codex-hud/internal/tui"
)

func RenderActivity(s *state.Session, width int) string {
	if len(s.ActiveTools) == 0 && len(s.ToolCounts) == 0 {
		return ""
	}

	var lines []string

	// Active tools
	for _, tool := range s.ActiveTools {
		elapsed := time.Since(tool.StartAt).Truncate(time.Second)
		active := lipgloss.NewStyle().Foreground(tui.ColorYellow).Render(
			fmt.Sprintf("▸ %s (%s)", tool.Name, elapsed))
		lines = append(lines, active)
	}

	// Tool counts summary
	type toolCount struct {
		name  string
		count int
	}
	var counts []toolCount
	for name, count := range s.ToolCounts {
		counts = append(counts, toolCount{name, count})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	var parts []string
	for _, tc := range counts {
		if len(parts) >= 4 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s ×%d", tc.name, tc.count))
	}
	if len(parts) > 0 {
		lines = append(lines, tui.LabelStyle.Render(strings.Join(parts, "  │  ")))
	}

	return tui.CardStyle.Width(width).Render(
		fmt.Sprintf("%s\n%s", tui.TitleStyle.Render("Activity"), strings.Join(lines, "\n")),
	)
}
```

**Step 3: 의존성 설치**

Run:
```bash
cd /Users/ds/codex-hud
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
```

**Step 4: 빌드 확인**

Run: `cd /Users/ds/codex-hud && go build ./internal/tui/... && go build ./internal/tui/components/...`
Expected: 빌드 성공 (에러 없음)

**Step 5: 커밋**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add TUI styles and card components (header, context, tokens, session, ratelimit, activity)"
```

---

### Task 8: TUI — bubbletea Model/Update/View

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/update.go`
- Create: `internal/tui/view.go`

**Step 1: model.go**

```go
// internal/tui/model.go
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ds/codex-hud/internal/config"
	gitpkg "github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/parser"
	"github.com/ds/codex-hud/internal/state"
)

// Msg types
type LineMsg string
type TickMsg time.Time
type GitStatusMsg *gitpkg.Status
type ErrorMsg error

// Model is the bubbletea model for the HUD.
type Model struct {
	State     *state.Session
	Config    *config.Config
	GitStatus *gitpkg.Status
	Lines     <-chan string
	Width     int
	Height    int
	Err       error
	Waiting   bool
}

func NewModel(cfg *config.Config, lines <-chan string) Model {
	return Model{
		State:   state.New(),
		Config:  cfg,
		Lines:   lines,
		Waiting: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForLine(m.Lines),
		tickCmd(),
	)
}

func waitForLine(lines <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if !ok {
			return nil
		}
		return LineMsg(line)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func fetchGitStatus(cwd string) tea.Cmd {
	return func() tea.Msg {
		status, err := gitpkg.GetStatus(cwd)
		if err != nil {
			return GitStatusMsg(nil)
		}
		return GitStatusMsg(status)
	}
}

func processLine(s *state.Session, line string) {
	event, err := parser.ParseLine(line)
	if err != nil {
		return
	}

	switch event.Type {
	case "session_meta":
		if meta, err := event.AsSessionMeta(); err == nil {
			s.ApplySessionMeta(meta)
		}
	case "turn_context":
		if tc, err := event.AsTurnContext(); err == nil {
			s.ApplyTurnContext(tc)
		}
	case "event_msg":
		if tc, err := event.AsTokenCount(); err == nil {
			s.ApplyTokenCount(tc)
		}
		// Check for task_started
		subtype, _ := event.EventMsgType()
		if subtype == "task_started" {
			s.IncrementTurn()
		}
	case "response_item":
		if fc, err := event.AsFunctionCall(); err == nil {
			s.ApplyFunctionCall(fc)
		}
		// Check for function_call_output to complete tool calls
		subtype, _ := event.EventMsgType()
		if subtype == "function_call_output" {
			// Extract tool_use_id to complete it
			// For now, we track by parsing the call_id from the output
		}
	}
}
```

Note: `EventMsgType()` needs to be exported from types.go. Add this to `internal/parser/types.go`:

```go
// EventMsgType returns the subtype of an event_msg or response_item event.
func (e *Event) EventMsgType() (string, error) {
	return e.eventMsgType()
}
```

**Step 2: update.go**

```go
// internal/tui/update.go
package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			// Force refresh git
			if m.State.CWD != "" {
				return m, fetchGitStatus(m.State.CWD)
			}
		case "m":
			// Toggle minimal mode (future)
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case LineMsg:
		m.Waiting = false
		processLine(m.State, string(msg))

		var cmds []tea.Cmd
		cmds = append(cmds, waitForLine(m.Lines))

		// Refresh git on turn_context (new CWD might have changed)
		if m.Config.Display.ShowGit && m.State.CWD != "" {
			cmds = append(cmds, fetchGitStatus(m.State.CWD))
		}
		return m, tea.Batch(cmds...)

	case TickMsg:
		return m, tickCmd()

	case GitStatusMsg:
		m.GitStatus = msg
	}

	return m, nil
}
```

**Step 3: view.go**

```go
// internal/tui/view.go
package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/tui/components"
)

func (m Model) View() string {
	if m.Waiting {
		return OuterStyle.Width(m.Width - 4).Render(
			TitleStyle.Render("codex-hud") + "\n\n" +
				LabelStyle.Render("Waiting for Codex session..."),
		)
	}

	w := m.Width - 8 // account for outer padding and borders
	if w < 40 {
		w = 40
	}

	// Header
	header := components.RenderHeader(m.State, w)

	// Context bar (full width)
	context := components.RenderContext(m.State, w)

	// Two columns: Tokens | Session
	halfW := (w - 3) / 2
	if m.Width < 80 {
		halfW = w // single column
	}

	tokens := components.RenderTokens(m.State, halfW)
	session := components.RenderSession(m.State, m.GitStatus, halfW)

	var middleRow string
	if m.Width >= 80 {
		middleRow = lipgloss.JoinHorizontal(lipgloss.Top, tokens, " ", session)
	} else {
		middleRow = tokens + "\n" + session
	}

	// Rate limit (conditional)
	rateLimit := ""
	if m.Config.Display.ShowRateLimit {
		rateLimit = components.RenderRateLimit(m.State, w)
	}

	// Activity (conditional)
	activity := ""
	if m.Config.Display.ShowActivity {
		activity = components.RenderActivity(m.State, w)
	}

	// Compose
	parts := []string{header, "", context, "", middleRow}
	if rateLimit != "" {
		parts = append(parts, "", rateLimit)
	}
	if activity != "" {
		parts = append(parts, "", activity)
	}

	inner := ""
	for i, part := range parts {
		if i > 0 {
			inner += "\n"
		}
		inner += part
	}

	title := fmt.Sprintf("─ %s ", TitleStyle.Render("codex-hud"))
	return OuterStyle.Width(m.Width - 2).Render(title + "\n" + inner)
}
```

**Step 4: 빌드 확인**

Run: `cd /Users/ds/codex-hud && go build ./internal/tui/...`
Expected: 빌드 성공

**Step 5: 커밋**

```bash
git add internal/tui/ internal/parser/types.go
git commit -m "feat: add bubbletea Model/Update/View with line processing and git refresh"
```

---

### Task 9: main.go — Watch 모드 구현

**Files:**
- Modify: `cmd/codex-hud/main.go`

**Step 1: cobra CLI + watch 모드 구현**

```go
// cmd/codex-hud/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ds/codex-hud/internal/config"
	"github.com/ds/codex-hud/internal/tui"
	"github.com/ds/codex-hud/internal/watcher"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "codex-hud",
		Short: "Real-time HUD for OpenAI Codex CLI",
		RunE:  runWatch, // Default: watch mode (wrapper mode TBD in Task 10)
	}

	rootCmd.Flags().String("file", "", "Path to a specific .jsonl session file")
	rootCmd.Flags().Bool("watch", false, "Watch mode: monitor existing session")
	rootCmd.Flags().StringP("version", "v", "", "Print version")

	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runWatch(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}

	configPath := filepath.Join(home, ".codex", "codex-hud.toml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	sessionsDir := filepath.Join(home, ".codex", "sessions")

	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		filePath, err = watcher.FindLatestSession(sessionsDir)
		if err != nil {
			// No session yet — start TUI in waiting mode and watch for new files
			filePath = ""
		}
	}

	lines := make(chan string, 100)
	stop := make(chan struct{})
	defer close(stop)

	if filePath != "" {
		go func() {
			if err := watcher.TailFile(filePath, lines, stop); err != nil {
				fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
			}
		}()
	} else {
		// Watch for new session files
		go func() {
			watcher.WatchForNewSession(sessionsDir, lines, stop)
		}()
	}

	model := tui.NewModel(cfg, lines)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
```

Note: `WatchForNewSession` 함수를 `internal/watcher/watcher.go`에 추가해야 함:

```go
// WatchForNewSession watches the sessions directory for new .jsonl files
// and starts tailing the newest one.
func WatchForNewSession(sessionsDir string, lines chan<- string, stop <-chan struct{}) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer w.Close()

	// Watch the sessions directory recursively
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			w.Add(path)
		}
		return nil
	})
	w.Add(sessionsDir)

	for {
		select {
		case <-stop:
			return
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create != 0 && strings.HasSuffix(event.Name, ".jsonl") {
				// New session file created — start tailing it
				go TailFile(event.Name, lines, stop)
				return
			}
			if event.Op&fsnotify.Create != 0 {
				// New directory — add to watcher
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.Add(event.Name)
				}
			}
		}
	}
}
```

**Step 2: 의존성 설치**

Run:
```bash
cd /Users/ds/codex-hud
go get github.com/spf13/cobra
go mod tidy
```

**Step 3: 빌드 + 실행 테스트**

Run:
```bash
cd /Users/ds/codex-hud
go build -o dist/codex-hud ./cmd/codex-hud
./dist/codex-hud --version
```
Expected: `codex-hud version 0.1.0`

Run: `./dist/codex-hud`
Expected: TUI가 뜨면서 실제 Codex 세션 데이터를 파싱하여 표시 (또는 "Waiting for Codex session...")

**Step 4: 커밋**

```bash
git add cmd/ internal/watcher/ go.mod go.sum
git commit -m "feat: implement watch mode with CLI, session auto-detect, and full TUI rendering"
```

---

### Task 10: Launcher — 환경 감지 + Wrapper 모드

**Files:**
- Create: `internal/launcher/launcher.go`
- Create: `internal/launcher/tmux.go`
- Create: `internal/launcher/wt.go`
- Create: `internal/launcher/fallback.go`
- Modify: `cmd/codex-hud/main.go` — wrapper 모드 추가

**Step 1: launcher.go — 환경 감지**

```go
// internal/launcher/launcher.go
package launcher

import (
	"os"
	"runtime"
)

type Environment int

const (
	EnvTmux Environment = iota
	EnvWindowsTerminal
	EnvGeneric
)

// Detect returns the current terminal environment.
func Detect() Environment {
	if os.Getenv("TMUX") != "" {
		return EnvTmux
	}
	if os.Getenv("WT_SESSION") != "" {
		return EnvWindowsTerminal
	}
	return EnvGeneric
}

// Launch starts codex in one pane and codex-hud --watch in another.
// The split method depends on the detected environment.
func Launch(codexArgs []string, split string, hudBinary string) error {
	env := Detect()
	switch env {
	case EnvTmux:
		return launchTmux(codexArgs, split, hudBinary)
	case EnvWindowsTerminal:
		return launchWT(codexArgs, split, hudBinary)
	default:
		return launchFallback(codexArgs, hudBinary, runtime.GOOS)
	}
}
```

**Step 2: tmux.go**

```go
// internal/launcher/tmux.go
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func launchTmux(codexArgs []string, split string, hudBinary string) error {
	direction := "-v" // vertical split (bottom)
	size := "30"
	if split == "right" {
		direction = "-h" // horizontal split
	}

	// Split current pane and run HUD in new pane
	hudCmd := fmt.Sprintf("%s --watch", hudBinary)
	splitCmd := exec.Command("tmux", "split-window", direction, "-l", size+"%", hudCmd)
	if err := splitCmd.Run(); err != nil {
		return fmt.Errorf("tmux split-window: %w", err)
	}

	// Run codex in original pane
	codexBin := "codex"
	cmd := exec.Command(codexBin, codexArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

**Step 3: wt.go**

```go
// internal/launcher/wt.go
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func launchWT(codexArgs []string, split string, hudBinary string) error {
	direction := "-V" // vertical (bottom)
	if split == "right" {
		direction = "-H" // horizontal
	}

	hudCmd := fmt.Sprintf("%s --watch", hudBinary)

	// Windows Terminal split-pane
	wtArgs := []string{"split-pane", direction, "--size", "0.3", hudCmd}
	cmd := exec.Command("wt", wtArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wt split-pane: %w", err)
	}

	// Run codex in current pane
	codexBin := "codex"
	codex := exec.Command(codexBin, codexArgs...)
	codex.Stdin = os.Stdin
	codex.Stdout = os.Stdout
	codex.Stderr = os.Stderr
	return codex.Run()
}
```

**Step 4: fallback.go**

```go
// internal/launcher/fallback.go
package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

func launchFallback(codexArgs []string, hudBinary string, goos string) error {
	// Launch HUD in a new terminal window
	hudCmd := fmt.Sprintf("%s --watch", hudBinary)

	switch goos {
	case "darwin":
		exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "Terminal" to do script "%s"`, hudCmd)).Run()
	case "linux":
		// Try common terminal emulators
		for _, term := range []string{"x-terminal-emulator", "gnome-terminal", "xterm"} {
			if _, err := exec.LookPath(term); err == nil {
				exec.Command(term, "-e", hudCmd).Start()
				break
			}
		}
	case "windows":
		exec.Command("cmd", "/c", "start", "cmd", "/k", hudCmd).Start()
	}

	// Run codex in current terminal
	codexBin := "codex"
	cmd := exec.Command(codexBin, codexArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

**Step 5: main.go에 wrapper 모드 추가**

`cmd/codex-hud/main.go`의 `rootCmd`를 수정하여 `--watch` 없이 실행하면 wrapper 모드로 동작:

```go
rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
    watch, _ := cmd.Flags().GetBool("watch")
    if watch {
        return runWatch(cmd, args)
    }
    return runWrapper(cmd, args)
}
```

```go
func runWrapper(cmd *cobra.Command, args []string) error {
    split, _ := cmd.Flags().GetString("split")
    if split == "" {
        split = "bottom"
    }

    self, err := os.Executable()
    if err != nil {
        self = "codex-hud"
    }

    // Everything after -- is passed to codex
    codexArgs := cmd.ArgsLenAtDash()
    var passthrough []string
    if codexArgs >= 0 {
        passthrough = args[codexArgs:]
    }

    return launcher.Launch(passthrough, split, self)
}
```

**Step 6: 빌드 확인**

Run: `cd /Users/ds/codex-hud && go build -o dist/codex-hud ./cmd/codex-hud`
Expected: 빌드 성공

**Step 7: 커밋**

```bash
git add internal/launcher/ cmd/codex-hud/main.go
git commit -m "feat: add launcher with env detection (tmux, Windows Terminal, fallback) and wrapper mode"
```

---

### Task 11: Integration Test — 실제 세션 데이터로 E2E 검증

**Files:**
- Create: `internal/parser/integration_test.go`

**Step 1: 실제 세션 파일 파싱 테스트**

```go
// internal/parser/integration_test.go
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

	// Verify first event is session_meta
	if events[0].Type != "session_meta" {
		t.Errorf("first event should be session_meta, got %s", events[0].Type)
	}

	// Count event types
	counts := make(map[string]int)
	for _, e := range events {
		counts[e.Type]++
	}
	t.Logf("Event type distribution: %v", counts)

	// Should have token_count events
	hasTokenCount := false
	for _, e := range events {
		if e.Type == "event_msg" {
			if tc, err := e.AsTokenCount(); err == nil && tc != nil {
				hasTokenCount = true
				t.Logf("Token count: in=%d, out=%d, context=%d",
					tc.Info.TotalTokenUsage.InputTokens,
					tc.Info.TotalTokenUsage.OutputTokens,
					tc.Info.ModelContextWindow)
				break
			}
		}
	}
	if !hasTokenCount {
		t.Log("Warning: no token_count events found (older session format?)")
	}
}
```

**Step 2: 테스트 실행**

Run: `cd /Users/ds/codex-hud && go test ./internal/parser/ -v -run TestParseRealSession`
Expected: PASS — 실제 세션 데이터 파싱 성공, 이벤트 분포 로그 출력

**Step 3: 커밋**

```bash
git add internal/parser/integration_test.go
git commit -m "test: add integration test with real Codex session data"
```

---

### Task 12: Cross-Platform 빌드 + 최종 확인

**Step 1: 전체 테스트 실행**

Run: `cd /Users/ds/codex-hud && go test ./... -v -timeout 60s`
Expected: ALL PASS

**Step 2: 크로스 플랫폼 빌드**

Run: `cd /Users/ds/codex-hud && make build-all`
Expected: `dist/` 디렉토리에 4개 바이너리 생성

**Step 3: macOS 바이너리 실행 테스트**

Run: `./dist/codex-hud-darwin-arm64 --version`
Expected: `codex-hud version 0.1.0`

Run: `./dist/codex-hud-darwin-arm64 --watch`
Expected: TUI 렌더링 + 실제 Codex 세션 데이터 표시

**Step 4: 최종 커밋**

```bash
git add -A
git commit -m "chore: finalize v0.1.0 with cross-platform build support"
```

---

## Task Summary

| Task | 내용 | 의존성 |
|------|------|--------|
| 0 | Go 설치 + 프로젝트 초기화 | - |
| 1 | JSONL Parser 타입 정의 | 0 |
| 2 | JSONL Parser 라인 파서 | 1 |
| 3 | State Store | 1 |
| 4 | File Watcher | 2 |
| 5 | Git Status | 0 |
| 6 | Config | 0 |
| 7 | TUI Styles + Components | 3, 5, 6 |
| 8 | TUI bubbletea Model/Update/View | 7 |
| 9 | main.go Watch 모드 | 4, 8 |
| 10 | Launcher (환경 감지 + Wrapper) | 9 |
| 11 | Integration Test | 9 |
| 12 | Cross-Platform 빌드 + 최종 확인 | 10, 11 |

병렬 가능: Task 1-2 / Task 3 / Task 5 / Task 6 은 서로 독립적이므로 동시 진행 가능.
