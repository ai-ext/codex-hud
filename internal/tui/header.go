package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ds/codex-hud/internal/state"
)

// RenderHeader renders the top header line showing model, effort, policy, and
// CLI version.
func RenderHeader(s *state.Session, width int) string {
	model := s.Model
	if model == "" {
		model = "unknown"
	}

	effort := s.ReasoningEffort
	if effort == "" {
		effort = "default"
	}

	policy := s.ApprovalPolicy
	if policy == "" {
		policy = "unknown"
	}

	version := s.CLIVersion
	if version == "" {
		version = "?"
	}

	dot := lipgloss.NewStyle().Foreground(ColorGreen).Render("●")
	modelStr := lipgloss.NewStyle().Bold(true).Foreground(ColorWhite).Render(model)
	effortStr := LabelStyle.Render(effort)
	policyStr := LabelStyle.Render(policy)
	versionStr := LabelStyle.Render(fmt.Sprintf("v%s", version))

	line := fmt.Sprintf("%s %s    %s    %s    %s", dot, modelStr, effortStr, policyStr, versionStr)

	return lipgloss.NewStyle().Width(width).Render(line)
}
