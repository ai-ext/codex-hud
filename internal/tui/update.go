package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and returns the updated model and any
// commands to execute.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			// Manual git refresh
			return m, fetchGitStatus(m.State.CWD)
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case LineMsg:
		m.Waiting = false
		processLine(m.State, string(msg))
		return m, tea.Batch(
			waitForLine(m.Lines),
			fetchGitStatus(m.State.CWD),
		)

	case TickMsg:
		return m, tickCmd()

	case GitStatusMsg:
		if msg.Status != nil {
			m.GitStatus = msg.Status
		}
		return m, nil
	}

	return m, nil
}
