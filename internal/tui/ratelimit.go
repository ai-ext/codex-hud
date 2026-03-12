package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderRateLimit renders the usage card showing primary (5h) and secondary (7d)
// rate limits side by side with progress bars. Returns empty string if no data.
func RenderRateLimit(s *state.Session, width int, minContentLines ...int) string {
	if !s.HasRateLimits {
		return ""
	}

	title := TitleStyle.Render("Usage")

	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	primaryLine := renderUsageLine(
		s.PrimaryWindowMinutes,
		s.PrimaryRatePercent,
		s.PrimaryResetsAt,
		innerWidth,
	)

	secondaryLine := renderUsageLine(
		s.SecondaryWindowMinutes,
		s.SecondaryRatePercent,
		s.SecondaryResetsAt,
		innerWidth,
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		primaryLine,
		secondaryLine,
	)

	// Pad to minimum content height if requested.
	if len(minContentLines) > 0 && minContentLines[0] > 0 {
		lines := lipgloss.Height(content)
		for lines < minContentLines[0] {
			content += "\n"
			lines++
		}
	}

	return CardStyle.Width(width - 2).Render(content)
}

// renderUsageLine renders a single usage row like:
//   5h  ██░░░░░░░░ 23%  resets in 4h 21m
func renderUsageLine(windowMinutes int, usedPercent float64, resetsAt int64, width int) string {
	windowLabel := formatWindow(windowMinutes)
	resetLabel := formatResetTime(resetsAt)
	pctStr := fmt.Sprintf("%.0f%%", usedPercent)

	// Layout: "5h  ████░░░░ 23%  resets in 4h 21m"
	// Reserve space for label + pct + reset text + padding
	labelWidth := len(windowLabel) + 2 // "5h  "
	pctWidth := len(pctStr) + 1        // " 23%"
	resetWidth := len(resetLabel) + 2   // "  resets in ..."
	barWidth := width - labelWidth - pctWidth - resetWidth
	if barWidth < 8 {
		barWidth = 8
	}

	color := BarColor(usedPercent, [2]float64{50, 80})
	bar := ProgressBar(barWidth, usedPercent, color)

	windowStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite).Width(labelWidth)
	pctStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	resetStyle := lipgloss.NewStyle().Foreground(ColorDim)

	return windowStyle.Render(windowLabel) +
		bar + " " +
		pctStyle.Render(pctStr) +
		resetStyle.Render("  "+resetLabel)
}

// formatWindow converts minutes into a human-readable window label.
// 300 -> "5h", 10080 -> "7d", 60 -> "1h", 1440 -> "1d"
func formatWindow(minutes int) string {
	if minutes <= 0 {
		return "?"
	}
	if minutes >= 1440 {
		days := minutes / 1440
		return fmt.Sprintf("%dd", days)
	}
	hours := minutes / 60
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatResetTime returns a human-readable "resets in Xh Ym" string.
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

	if hours >= 24 {
		days := hours / 24
		h := hours % 24
		return fmt.Sprintf("resets in %dd %dh", days, h)
	}
	if hours > 0 {
		return fmt.Sprintf("resets in %dh %dm", hours, minutes)
	}
	return fmt.Sprintf("resets in %dm", minutes)
}
