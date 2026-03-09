package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderTokens renders the token usage card showing input, cached, output,
// and reasoning token counts.
func RenderTokens(s *state.Session, width int) string {
	title := TitleStyle.Render("Tokens")

	rows := []string{
		tokenRow("↓", "in", s.TotalInputTokens),
		tokenRow("↻", "cache", s.TotalCachedTokens),
		tokenRow("↑", "out", s.TotalOutputTokens),
		tokenRow("◆", "reason", s.TotalReasonTokens),
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title}, rows...)...,
	)

	return CardStyle.Width(width - 2).Render(content)
}

// tokenRow formats a single token row with icon, label, and count.
func tokenRow(icon, label string, count int) string {
	iconStr := lipgloss.NewStyle().Foreground(ColorCyan).Render(icon)
	labelStr := LabelStyle.Render(label)
	valueStr := ValueStyle.Render(FormatNumber(count))
	return fmt.Sprintf("%s %s %s", iconStr, labelStr, valueStr)
}
