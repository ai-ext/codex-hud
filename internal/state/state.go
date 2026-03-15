// Package state aggregates parsed JSONL events into a single session state.
package state

import (
	"regexp"
	"strings"
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
	HasRateLimits           bool
	PrimaryRatePercent      float64
	PrimaryResetsAt         int64
	PrimaryWindowMinutes    int
	SecondaryRatePercent    float64
	SecondaryResetsAt       int64
	SecondaryWindowMinutes  int

	// Turn tracking
	TurnCount int

	// Tool tracking
	ToolCounts  map[string]int
	ActiveTools []ActiveTool

	// Skill tracking
	SkillRegistry    map[string]string // file path → skill name
	ActiveSkillNames []string          // activated skill names (deduplicated)
}

// New creates a new Session with initialized maps and slices.
func New() *Session {
	return &Session{
		ToolCounts:       make(map[string]int),
		ActiveTools:      make([]ActiveTool, 0),
		SkillRegistry:    make(map[string]string),
		ActiveSkillNames: make([]string, 0),
	}
}

// ApplySessionMeta sets session-level metadata from a parsed SessionMeta
// event. If the session ID changes (i.e. a new Codex session started), all
// session-specific state is reset while preserving account-level data such as
// rate limits (which come from the WHAM API, not the session file).
func (s *Session) ApplySessionMeta(m *parser.SessionMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Detect session switch and reset per-session state.
	// Rate limits (from WHAM API) are account-level and preserved.
	if s.SessionID != "" && s.SessionID != m.ID {
		s.CLIVersion = ""
		s.CWD = ""
		s.ModelProvider = ""
		s.StartTime = time.Time{}
		s.Model = ""
		s.ReasoningEffort = ""
		s.ApprovalPolicy = ""
		s.SandboxType = ""
		s.ContextWindowSize = 0
		s.ContextUsedTokens = 0
		s.TotalInputTokens = 0
		s.TotalCachedTokens = 0
		s.TotalOutputTokens = 0
		s.TotalReasonTokens = 0
		s.TurnCount = 0
		s.ToolCounts = make(map[string]int)
		s.ActiveTools = make([]ActiveTool, 0)
		s.SkillRegistry = make(map[string]string)
		s.ActiveSkillNames = make([]string, 0)
	}

	s.SessionID = m.ID
	s.CLIVersion = m.CLIVersion
	s.CWD = m.CWD
	s.ModelProvider = m.ModelProvider

	if t, err := time.Parse(time.RFC3339Nano, m.Timestamp); err == nil {
		s.StartTime = t
	}
}

// skillLineRe matches skill entries in user_instructions:
//   - skill-name: description (file: /path/to/skill)
var skillLineRe = regexp.MustCompile(`- ([^:]+): .+ \(file: ([^)]+)\)`)

// ApplyTurnContext sets model and policy information from a parsed
// TurnContext event.
func (s *Session) ApplyTurnContext(tc *parser.TurnContext) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Model = tc.Model
	s.ReasoningEffort = tc.CollaborationMode.Settings.ReasoningEffort
	s.ApprovalPolicy = tc.ApprovalPolicy
	s.SandboxType = tc.SandboxPolicy.Type

	// Clear active skills each turn — only skills accessed in the current
	// turn will be shown (real-time display).
	s.ActiveSkillNames = s.ActiveSkillNames[:0]

	// Parse skill registry from user_instructions.
	if tc.UserInstructions != "" {
		matches := skillLineRe.FindAllStringSubmatch(tc.UserInstructions, -1)
		for _, m := range matches {
			name := strings.TrimSpace(m[1])
			path := strings.TrimSpace(m[2])
			s.SkillRegistry[path] = name
		}
	}
}

// ApplyUserMessage extracts skill invocations from the user's message.
// Skill references appear as $skill-name placeholders in text_elements.
func (s *Session) ApplyUserMessage(um *parser.UserMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, te := range um.TextElements {
		if strings.HasPrefix(te.Placeholder, "$") {
			name := strings.TrimPrefix(te.Placeholder, "$")
			if name != "" {
				s.addActiveSkill(name)
			}
		}
	}
}

// addActiveSkill appends a skill name if not already present.
// Must be called with s.mu held.
func (s *Session) addActiveSkill(name string) {
	for _, existing := range s.ActiveSkillNames {
		if existing == name {
			return
		}
	}
	s.ActiveSkillNames = append(s.ActiveSkillNames, name)
}

// ApplyTokenCount sets token usage, context window, and rate limit data from
// a parsed TokenCount event.
func (s *Session) ApplyTokenCount(tc *parser.TokenCount) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only update token data if info is present (some events have info:null).
	if tc.Info.ModelContextWindow > 0 || tc.Info.TotalTokenUsage.TotalTokens > 0 {
		totalUsage := tc.Info.TotalTokenUsage
		lastUsage := tc.Info.LastTokenUsage
		s.TotalInputTokens = totalUsage.InputTokens
		s.TotalCachedTokens = totalUsage.CachedInputTokens
		s.TotalOutputTokens = totalUsage.OutputTokens
		s.TotalReasonTokens = totalUsage.ReasoningOutputTokens

		s.ContextWindowSize = tc.Info.ModelContextWindow

		// Prefer the latest turn usage for the context card because the total
		// session usage can grow far beyond the active model context window.
		if lastUsage.TotalTokens > 0 {
			s.ContextUsedTokens = lastUsage.TotalTokens
		} else {
			s.ContextUsedTokens = totalUsage.TotalTokens
		}
	}

	// Rate limits are no longer processed from session data. They are fetched
	// exclusively via the WHAM /usage API (internal/usage package) to avoid
	// stale data from old sessions flashing on startup.
}

// ContextPercent returns the percentage of the context window currently in
// use. Returns 0 if the context window size is zero.
func (s *Session) ContextPercent() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ContextWindowSize == 0 {
		return 0
	}
	pct := float64(s.ContextUsedTokens) / float64(s.ContextWindowSize) * 100.0
	if pct > 100 {
		pct = 100
	}
	return pct
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
