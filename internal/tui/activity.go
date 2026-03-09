package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderActivity renders the activity card showing currently active tools and
// top tool usage counts. Returns an empty string if no tools have been used.
func RenderActivity(s *state.Session, width int) string {
	if len(s.ToolCounts) == 0 && len(s.ActiveTools) == 0 {
		return ""
	}

	title := TitleStyle.Render("Activity")

	var rows []string

	// Active tools (shown in yellow)
	for _, at := range s.ActiveTools {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(ColorYellow).Render(
				fmt.Sprintf("▶ %s", at.Name),
			),
		)
	}

	// Top-4 tool counts sorted by count descending
	type toolEntry struct {
		name  string
		count int
	}

	entries := make([]toolEntry, 0, len(s.ToolCounts))
	for name, count := range s.ToolCounts {
		entries = append(entries, toolEntry{name: name, count: count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].name < entries[j].name
	})

	limit := 4
	if len(entries) < limit {
		limit = len(entries)
	}
	for _, e := range entries[:limit] {
		rows = append(rows, fmt.Sprintf("%s %s",
			LabelStyle.Render(e.name),
			ValueStyle.Render(fmt.Sprintf("x%d", e.count)),
		))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title}, rows...)...,
	)

	return CardStyle.Width(width - 2).Render(content)
}
