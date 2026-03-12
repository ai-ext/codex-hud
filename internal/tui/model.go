package tui

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ds/codex-hud/internal/config"
	"github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/parser"
	"github.com/ds/codex-hud/internal/state"
)

// Model is the top-level bubbletea model for the codex-hud TUI.
type Model struct {
	State     *state.Session
	Config    *config.Config
	GitStatus *git.Status
	Lines     <-chan string
	Width     int
	Height    int
	Err       error
	Waiting   bool
}

// LineMsg is sent when a new JSONL line is read from the session file.
type LineMsg string

// TickMsg is sent periodically to update time-dependent displays.
type TickMsg time.Time

// GitStatusMsg is sent when a git status fetch completes.
type GitStatusMsg struct {
	Status *git.Status
}

// NewModel creates a new Model with default state and the given config and
// line channel.
func NewModel(cfg *config.Config, lines <-chan string) Model {
	return Model{
		State:   state.New(),
		Config:  cfg,
		Lines:   lines,
		Width:   80,
		Height:  24,
		Waiting: true,
	}
}

// Init returns the initial commands: wait for a line and start the tick loop.
func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForLine(m.Lines), tickCmd())
}

// waitForLine returns a Cmd that blocks until a line arrives on the channel,
// then sends it as a LineMsg.
func waitForLine(lines <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if !ok {
			return nil
		}
		return LineMsg(line)
	}
}

// tickCmd returns a Cmd that sends a TickMsg after 500ms.
func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// fetchGitStatus returns a Cmd that runs git.GetStatus and sends the result
// as a GitStatusMsg.
func fetchGitStatus(cwd string) tea.Cmd {
	return func() tea.Msg {
		if cwd == "" {
			return GitStatusMsg{}
		}
		s, err := git.GetStatus(cwd)
		if err != nil {
			return GitStatusMsg{}
		}
		return GitStatusMsg{Status: s}
	}
}

// processLine parses a JSONL line and applies it to the session state.
func processLine(s *state.Session, line string) {
	ev, err := parser.ParseLine(line)
	if err != nil {
		return
	}

	switch ev.Type {
	case "session_meta":
		m, err := ev.AsSessionMeta()
		if err != nil {
			return
		}
		s.ApplySessionMeta(m)

	case "turn_context":
		tc, err := ev.AsTurnContext()
		if err != nil {
			return
		}
		s.ApplyTurnContext(tc)
		s.IncrementTurn()

	case "event_msg":
		subtype, err := ev.EventMsgType()
		if err != nil {
			return
		}
		switch subtype {
		case "token_count":
			tc, err := ev.AsTokenCount()
			if err != nil {
				return
			}
			s.ApplyTokenCount(tc)
		case "task_started":
			// Task started events are noted but no specific state update needed.
		}

	case "response_item":
		// Peek at the inner type/subtype field to determine how to handle the
		// response item. Newer Codex CLI uses "type", older uses "subtype".
		var env struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		}
		if err := json.Unmarshal(ev.Payload, &env); err != nil {
			return
		}
		itemType := env.Subtype
		if itemType == "" {
			itemType = env.Type
		}

		switch itemType {
		case "function_call":
			fc, err := ev.AsFunctionCall()
			if err != nil {
				return
			}
			s.ApplyFunctionCall(fc)

		case "function_call_output":
			// Extract the call_id to complete the function call.
			var output struct {
				CallID string `json:"call_id"`
			}
			if err := json.Unmarshal(ev.Payload, &output); err != nil {
				return
			}
			if output.CallID != "" {
				s.CompleteFunctionCall(output.CallID)
			}
		}
	}
}
