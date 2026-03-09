package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI layout.
func (m Model) View() string {
	if m.Waiting {
		msg := lipgloss.NewStyle().
			Foreground(ColorDim).
			Render("Waiting for Codex session...")
		return OuterStyle.Width(m.Width - 4).Render(msg)
	}

	// Compute available inner width (outer border + padding: 2 border + 4 padding = 6)
	innerWidth := m.Width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var sections []string

	// Header
	sections = append(sections, RenderHeader(m.State, innerWidth))

	// Context bar
	sections = append(sections, RenderContext(m.State, innerWidth))

	// Tokens and Session side-by-side if wide enough
	if m.Width >= 80 {
		halfWidth := innerWidth / 2
		tokensCard := RenderTokens(m.State, halfWidth)
		sessionCard := RenderSession(m.State, m.GitStatus, halfWidth)
		row := lipgloss.JoinHorizontal(lipgloss.Top, tokensCard, sessionCard)
		sections = append(sections, row)
	} else {
		sections = append(sections, RenderTokens(m.State, innerWidth))
		sections = append(sections, RenderSession(m.State, m.GitStatus, innerWidth))
	}

	// Rate limit (conditional)
	if m.Config.Display.ShowRateLimit {
		if rl := RenderRateLimit(m.State, innerWidth); rl != "" {
			sections = append(sections, rl)
		}
	}

	// Activity (conditional)
	if m.Config.Display.ShowActivity {
		if act := RenderActivity(m.State, innerWidth); act != "" {
			sections = append(sections, act)
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return OuterStyle.Width(m.Width - 4).Render(body)
}
