package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// usageRefreshMsg triggers a new usage API fetch.
type usageRefreshMsg struct{}

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
		ProcessLine(m.State, string(msg))
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

	case UsageMsg:
		if msg.Response != nil && msg.Response.RateLimit != nil {
			rl := msg.Response.RateLimit
			m.State.HasRateLimits = true
			if rl.Primary != nil {
				m.State.PrimaryRatePercent = float64(rl.Primary.UsedPercent)
				m.State.PrimaryResetsAt = int64(rl.Primary.ResetAt)
				m.State.PrimaryWindowMinutes = rl.Primary.LimitWindowSecs / 60
			}
			if rl.Secondary != nil {
				m.State.SecondaryRatePercent = float64(rl.Secondary.UsedPercent)
				m.State.SecondaryResetsAt = int64(rl.Secondary.ResetAt)
				m.State.SecondaryWindowMinutes = rl.Secondary.LimitWindowSecs / 60
			}
		}
		// Re-fetch usage every 30 seconds.
		return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
			return usageRefreshMsg{}
		})

	case usageRefreshMsg:
		return m, fetchUsage()
	}

	return m, nil
}
