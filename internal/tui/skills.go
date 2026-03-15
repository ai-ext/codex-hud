package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderSkills renders the skills card showing activated skills.
// Returns an empty string if no skills have been activated.
func RenderSkills(s *state.Session, width int) string {
	if len(s.ActiveSkillNames) == 0 {
		return ""
	}

	title := TitleStyle.Render("Skills")

	var rows []string
	for _, name := range s.ActiveSkillNames {
		rows = append(rows,
			fmt.Sprintf("%s %s",
				lipgloss.NewStyle().Foreground(ColorYellow).Render("◆"),
				ValueStyle.Render(name),
			),
		)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title}, rows...)...,
	)

	return CardStyle.Width(width - 2).Render(content)
}

// RenderSkillsCompact renders activated skills in a single line with title.
// Returns an empty string if no skills have been activated.
func RenderSkillsCompact(s *state.Session, width int) string {
	if len(s.ActiveSkillNames) == 0 {
		return ""
	}

	var parts []string
	for _, name := range s.ActiveSkillNames {
		parts = append(parts,
			fmt.Sprintf("%s %s",
				lipgloss.NewStyle().Foreground(ColorYellow).Render("◆"),
				ValueStyle.Render(name),
			),
		)
	}

	return TitleStyle.Render("Skills") + "  " + strings.Join(parts, "  ")
}
