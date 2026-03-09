// Package state aggregates parsed JSONL events into a single session state.
package state

import (
	"sync"
	"time"

	"github.com/ds/codex-hud/internal/parser"
)

// ActiveTool represents a tool invocation that has started but not yet
// completed.
type ActiveTool struct {
	Name    string
	CallID  string
	StartAt time.Time
}

// Session holds the aggregated state of a Codex CLI session, built
// incrementally by applying parsed events.
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
	TotalInputTokens  int
	TotalCachedTokens int
	TotalOutputTokens int
	TotalReasonTokens int

	// Rate limits
	HasRateLimits        bool
	PrimaryRatePercent   float64
	PrimaryResetsAt      int64
	SecondaryRatePercent float64
	SecondaryResetsAt    int64

	// Turn tracking
	TurnCount int

	// Tool tracking
	ToolCounts  map[string]int
	ActiveTools []ActiveTool
}

// New creates a new Session with initialized maps and slices.
func New() *Session {
	return &Session{
		ToolCounts:  make(map[string]int),
		ActiveTools: make([]ActiveTool, 0),
	}
}

// ApplySessionMeta sets session-level metadata from a parsed SessionMeta
// event.
func (s *Session) ApplySessionMeta(m *parser.SessionMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SessionID = m.ID
	s.CLIVersion = m.CLIVersion
	s.CWD = m.CWD
	s.ModelProvider = m.ModelProvider

	if t, err := time.Parse(time.RFC3339Nano, m.Timestamp); err == nil {
		s.StartTime = t
	}
}

// ApplyTurnContext sets model and policy information from a parsed
// TurnContext event.
func (s *Session) ApplyTurnContext(tc *parser.TurnContext) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Model = tc.Model
	s.ReasoningEffort = tc.CollaborationMode.Settings.ReasoningEffort
	s.ApprovalPolicy = tc.ApprovalPolicy
	s.SandboxType = tc.SandboxPolicy.Type
}

// ApplyTokenCount sets token usage, context window, and rate limit data from
// a parsed TokenCount event.
func (s *Session) ApplyTokenCount(tc *parser.TokenCount) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usage := tc.Info.TotalTokenUsage
	s.TotalInputTokens = usage.InputTokens
	s.TotalCachedTokens = usage.CachedInputTokens
	s.TotalOutputTokens = usage.OutputTokens
	s.TotalReasonTokens = usage.ReasoningOutputTokens

	s.ContextWindowSize = tc.Info.ModelContextWindow
	s.ContextUsedTokens = usage.TotalTokens

	if tc.RateLimits != nil {
		s.HasRateLimits = true
		s.PrimaryRatePercent = tc.RateLimits.Primary.UsedPercent
		s.PrimaryResetsAt = tc.RateLimits.Primary.ResetsAt
		s.SecondaryRatePercent = tc.RateLimits.Secondary.UsedPercent
		s.SecondaryResetsAt = tc.RateLimits.Secondary.ResetsAt
	} else {
		s.HasRateLimits = false
	}
}

// ContextPercent returns the percentage of the context window currently in
// use. Returns 0 if the context window size is zero.
func (s *Session) ContextPercent() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ContextWindowSize == 0 {
		return 0
	}
	return float64(s.ContextUsedTokens) / float64(s.ContextWindowSize) * 100.0
}

// ApplyFunctionCall records a new tool invocation: increments the tool count
// and adds an entry to the active tools list.
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

// CompleteFunctionCall removes the tool invocation with the given callID from
// the active tools list. The cumulative tool count is not modified.
func (s *Session) CompleteFunctionCall(callID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, at := range s.ActiveTools {
		if at.CallID == callID {
			s.ActiveTools = append(s.ActiveTools[:i], s.ActiveTools[i+1:]...)
			return
		}
	}
}

// IncrementTurn increments the turn counter by one.
func (s *Session) IncrementTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TurnCount++
}
