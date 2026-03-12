package tui

import (
	"strings"

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

	// Compute available inner width.
	// OuterStyle uses Width(m.Width-4) which includes padding but not border.
	// Horizontal padding is 2*2=4, so content area = (m.Width-4) - 4 = m.Width-8.
	innerWidth := m.Width - 8
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

		// First pass: render both to measure content lines.
		tokensCard := RenderTokens(m.State, halfWidth)
		sessionCard := RenderSession(m.State, m.GitStatus, halfWidth)

		tLines := strings.Count(tokensCard, "\n") + 1
		sLines := strings.Count(sessionCard, "\n") + 1

		// Second pass: re-render with equalized content height.
		// The card has 2 border lines (top + bottom), so inner content
		// lines = total lines - 2. We want both to have the same total.
		maxLines := tLines
		if sLines > maxLines {
			maxLines = sLines
		}
		// inner content lines = maxLines - 2 (border top/bottom)
		minContent := maxLines - 2
		tokensCard = RenderTokens(m.State, halfWidth, minContent)
		sessionCard = RenderSession(m.State, m.GitStatus, halfWidth, minContent)

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
