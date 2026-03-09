package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderContext renders the context window usage card with a progress bar,
// percentage, and token counts.
func RenderContext(s *state.Session, width int) string {
	pct := s.ContextPercent()
	color := BarColor(pct, [2]float64{50, 75})

	title := TitleStyle.Render("Context")

	// Account for card padding (1 on each side) and border (1 on each side) = 4 total
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	bar := ProgressBar(innerWidth, pct, color)

	pctStr := lipgloss.NewStyle().Foreground(color).Bold(true).Render(
		fmt.Sprintf("%.1f%%", pct),
	)

	tokens := LabelStyle.Render(
		fmt.Sprintf("%s / %s tokens",
			FormatNumber(s.ContextUsedTokens),
			FormatNumber(s.ContextWindowSize),
		),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		bar,
		fmt.Sprintf("%s  %s", pctStr, tokens),
	)

	return CardStyle.Width(width - 2).Render(content)
}
