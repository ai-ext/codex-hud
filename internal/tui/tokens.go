package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderTokens renders the token usage card showing input, cached, output,
// and reasoning token counts. If minContentLines > 0, blank lines are appended
// to the content so the card's interior reaches at least that many lines.
func RenderTokens(s *state.Session, width int, minContentLines ...int) string {
	title := TitleStyle.Render("Tokens")

	rows := []string{
		tokenRow("↓", "in", s.TotalInputTokens),
		tokenRow("↻", "cache", s.TotalCachedTokens),
		tokenRow("↑", "out", s.TotalOutputTokens),
		tokenRow("◆", "reason", s.TotalReasonTokens),
	}

	all := append([]string{title}, rows...)
	if len(minContentLines) > 0 && minContentLines[0] > len(all) {
		for len(all) < minContentLines[0] {
			all = append(all, "")
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, all...)

	return CardStyle.Width(width - 2).Render(content)
}

// tokenRow formats a single token row with icon, label, and count.
func tokenRow(icon, label string, count int) string {
	iconStr := lipgloss.NewStyle().Foreground(ColorCyan).Render(icon)
	labelStr := LabelStyle.Render(label)
	valueStr := ValueStyle.Render(FormatNumber(count))
	return fmt.Sprintf("%s %s %s", iconStr, labelStr, valueStr)
}
