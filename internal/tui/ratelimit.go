package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderRateLimit renders the rate limit card showing primary rate usage with
// a progress bar, percentage, and reset time. Returns an empty string if no
// rate limit data is available.
func RenderRateLimit(s *state.Session, width int) string {
	if !s.HasRateLimits {
		return ""
	}

	title := TitleStyle.Render("Rate Limit")

	pct := s.PrimaryRatePercent
	color := BarColor(pct, [2]float64{50, 75})

	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	bar := ProgressBar(innerWidth, pct, color)

	pctStr := lipgloss.NewStyle().Foreground(color).Bold(true).Render(
		fmt.Sprintf("%.0f%%", pct),
	)

	resetStr := formatResetTime(s.PrimaryResetsAt)
	resetLabel := LabelStyle.Render(resetStr)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		bar,
		fmt.Sprintf("%s  %s", pctStr, resetLabel),
	)

	return CardStyle.Width(width - 2).Render(content)
}

// formatResetTime returns a human-readable "resets in Xh Ym" string for a
// Unix timestamp (seconds). If the timestamp is in the past, returns
// "resets now".
func formatResetTime(unixSec int64) string {
	if unixSec == 0 {
		return ""
	}

	resetAt := time.Unix(unixSec, 0)
	remaining := time.Until(resetAt)

	if remaining <= 0 {
		return "resets now"
	}

	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("resets in %dh %dm", hours, minutes)
	}
	return fmt.Sprintf("resets in %dm", minutes)
}
