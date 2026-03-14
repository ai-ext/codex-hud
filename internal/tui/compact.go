package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/state"
)

// RenderContextCompact renders context as a single line:
//   Context ██████░░░░ 44.5%  114,867 / 258,400
func RenderContextCompact(s *state.Session, width int) string {
	pct := s.ContextPercent()
	color := BarColor(pct, [2]float64{50, 75})

	label := TitleStyle.Render("Context")
	pctStr := lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%.1f%%", pct))
	tokens := LabelStyle.Render(fmt.Sprintf("%s / %s",
		FormatNumber(s.ContextUsedTokens),
		FormatNumber(s.ContextWindowSize),
	))

	// Calculate bar width from remaining space
	suffix := fmt.Sprintf(" %s  %s", pctStr, tokens)
	barWidth := width - lipgloss.Width(label) - lipgloss.Width(suffix) - 1
	if barWidth < 8 {
		barWidth = 8
	}

	bar := ProgressBar(barWidth, pct, color)
	return fmt.Sprintf("%s %s%s", label, bar, suffix)
}

// RenderTokensCompact renders all tokens in one line:
//   ↓in 2.6M  ↻cache 2.2M  ↑out 14K  ◆reason 5K
func RenderTokensCompact(s *state.Session, width int) string {
	parts := []string{
		fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(ColorCyan).Render("↓"),
			ValueStyle.Render(FormatNumber(s.TotalInputTokens))),
		fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(ColorCyan).Render("↻"),
			ValueStyle.Render(FormatNumber(s.TotalCachedTokens))),
		fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(ColorCyan).Render("↑"),
			ValueStyle.Render(FormatNumber(s.TotalOutputTokens))),
		fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(ColorCyan).Render("◆"),
			ValueStyle.Render(FormatNumber(s.TotalReasonTokens))),
	}
	return strings.Join(parts, "  ")
}

// RenderSessionCompact renders session info in one line:
//   ⏱ 20m 0s  turns 3  cwd ~  ⎇ main●
func RenderSessionCompact(s *state.Session, gitStatus *git.Status, width int) string {
	dur := formatDuration(time.Since(s.StartTime))
	cwd := shortenPath(s.CWD)

	parts := []string{
		fmt.Sprintf("%s %s",
			LabelStyle.Render("⏱"),
			ValueStyle.Render(dur)),
		fmt.Sprintf("%s %s",
			LabelStyle.Render("turns"),
			ValueStyle.Render(fmt.Sprintf("%d", s.TurnCount))),
	}

	if cwd != "" {
		parts = append(parts, fmt.Sprintf("%s %s",
			LabelStyle.Render("cwd"),
			ValueStyle.Render(cwd)))
	}

	if gitStatus != nil {
		branch := lipgloss.NewStyle().Foreground(ColorCyan).Render(gitStatus.Branch)
		gitStr := fmt.Sprintf("⎇ %s", branch)
		if gitStatus.Dirty {
			gitStr += lipgloss.NewStyle().Foreground(ColorYellow).Render("●")
		}
		parts = append(parts, gitStr)
	}

	return strings.Join(parts, "  ")
}

// RenderRateLimitCompact renders usage in 2 lines without card border:
//   Usage
//   5h ████░░ 1%  resets 4h 58m  │  7d ████████░░ 13%  resets 4d 19h
func RenderRateLimitCompact(s *state.Session, width int) string {
	if !s.HasRateLimits {
		return ""
	}

	title := TitleStyle.Render("Usage")

	pLine := renderUsageCompactItem(s.PrimaryWindowMinutes, s.PrimaryRatePercent, s.PrimaryResetsAt)
	sLine := renderUsageCompactItem(s.SecondaryWindowMinutes, s.SecondaryRatePercent, s.SecondaryResetsAt)

	sep := LabelStyle.Render(" │ ")
	line := pLine + sep + sLine

	return title + "\n" + line
}

func renderUsageCompactItem(windowMinutes int, usedPercent float64, resetsAt int64) string {
	windowLabel := formatWindow(windowMinutes)
	pctStr := fmt.Sprintf("%.0f%%", usedPercent)
	resetLabel := formatResetTime(resetsAt)

	color := BarColor(usedPercent, [2]float64{50, 80})
	bar := ProgressBar(10, usedPercent, color)

	windowStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	pctStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	resetStyle := lipgloss.NewStyle().Foreground(ColorDim)

	return windowStyle.Render(windowLabel) + " " + bar + " " +
		pctStyle.Render(pctStr) + " " +
		resetStyle.Render(resetLabel)
}

// RenderActivityCompact renders activity in one line:
//   ▶ exec_command  (exec_command x53, read x12)
func RenderActivityCompact(s *state.Session, width int) string {
	if len(s.ToolCounts) == 0 && len(s.ActiveTools) == 0 {
		return ""
	}

	var parts []string

	for _, at := range s.ActiveTools {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("▶ %s", at.Name)))
	}

	// Top 3 tool counts
	type toolEntry struct {
		name  string
		count int
	}
	entries := make([]toolEntry, 0, len(s.ToolCounts))
	for name, count := range s.ToolCounts {
		entries = append(entries, toolEntry{name, count})
	}
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].count > entries[i].count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	limit := 3
	if len(entries) < limit {
		limit = len(entries)
	}
	for _, e := range entries[:limit] {
		parts = append(parts, fmt.Sprintf("%s %s",
			LabelStyle.Render(e.name),
			ValueStyle.Render(fmt.Sprintf("x%d", e.count))))
	}

	return strings.Join(parts, "  ")
}
