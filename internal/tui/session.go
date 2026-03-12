package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/git"
	"github.com/ds/codex-hud/internal/state"
)

// RenderSession renders the session info card showing duration, turns, CWD,
// and git status. If minContentLines > 0, blank lines are appended to the
// content so the card's interior reaches at least that many lines.
func RenderSession(s *state.Session, gitStatus *git.Status, width int, minContentLines ...int) string {
	title := TitleStyle.Render("Session")

	var rows []string

	// Duration
	dur := formatDuration(time.Since(s.StartTime))
	rows = append(rows, infoRow("duration", dur))

	// Turns
	rows = append(rows, infoRow("turns", fmt.Sprintf("%d", s.TurnCount)))

	// CWD (shortened with ~)
	cwd := shortenPath(s.CWD)
	rows = append(rows, infoRow("cwd", cwd))

	// Git info
	if gitStatus != nil {
		branch := lipgloss.NewStyle().Foreground(ColorCyan).Render(gitStatus.Branch)
		gitLine := fmt.Sprintf("⎇ %s", branch)
		if gitStatus.Dirty {
			gitLine += lipgloss.NewStyle().Foreground(ColorYellow).Render(" ●")
		}
		rows = append(rows, gitLine)

		if gitStatus.Modified > 0 || gitStatus.Added > 0 || gitStatus.Deleted > 0 || gitStatus.Untracked > 0 {
			var stats []string
			if gitStatus.Modified > 0 {
				stats = append(stats, fmt.Sprintf("~%d", gitStatus.Modified))
			}
			if gitStatus.Added > 0 {
				stats = append(stats, fmt.Sprintf("+%d", gitStatus.Added))
			}
			if gitStatus.Deleted > 0 {
				stats = append(stats, fmt.Sprintf("-%d", gitStatus.Deleted))
			}
			if gitStatus.Untracked > 0 {
				stats = append(stats, fmt.Sprintf("?%d", gitStatus.Untracked))
			}
			rows = append(rows, LabelStyle.Render(strings.Join(stats, " ")))
		}
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

// infoRow formats a label: value row.
func infoRow(label, value string) string {
	return fmt.Sprintf("%s %s",
		LabelStyle.Render(label),
		ValueStyle.Render(value),
	)
}

// shortenPath replaces the user's home directory prefix with ~.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// formatDuration formats a duration as a human-readable string (e.g. "5m 12s").
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
