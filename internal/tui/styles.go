// Package tui provides the terminal user interface for codex-hud using
// bubbletea and lipgloss.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color constants (ANSI 256-color palette indices).
var (
	ColorCyan   = lipgloss.Color("86")
	ColorGreen  = lipgloss.Color("78")
	ColorYellow = lipgloss.Color("220")
	ColorRed    = lipgloss.Color("196")
	ColorDim    = lipgloss.Color("240")
	ColorWhite  = lipgloss.Color("255")
	ColorBorder = lipgloss.Color("240")
)

// Reusable styles.
var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	LabelStyle = lipgloss.NewStyle().Foreground(ColorDim)
	ValueStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	CardStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)
	OuterStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorCyan).
			Padding(1, 2)
)

// ProgressBar renders a horizontal bar of the given width filled to percent
// (0-100). Filled segments use the provided color; empty segments are dim.
func ProgressBar(width int, percent float64, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(float64(width) * percent / 100.0)
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorDim)

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

// BarColor returns green, yellow, or red depending on where percent falls
// relative to the two thresholds. percent < thresholds[0] => green,
// percent < thresholds[1] => yellow, else red.
func BarColor(percent float64, thresholds [2]float64) lipgloss.Color {
	if percent < thresholds[0] {
		return ColorGreen
	}
	if percent < thresholds[1] {
		return ColorYellow
	}
	return ColorRed
}

// FormatNumber formats an integer for display. Values >= 1,000,000 are shown
// as e.g. "1.4M", values >= 1,000 are shown with a comma (e.g. "1,385"),
// otherwise the plain number is returned.
func FormatNumber(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
	}
	if n >= 1_000 {
		return formatWithCommas(n)
	}
	return fmt.Sprintf("%d", n)
}

// formatWithCommas inserts comma separators into n (assumed positive).
func formatWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var b strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
