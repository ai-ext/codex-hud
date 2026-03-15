package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI layout.
func (m Model) View() string {
	if m.Waiting {
		innerWidth := m.Width - 8
		if innerWidth < 20 {
			innerWidth = 20
		}

		var parts []string
		parts = append(parts, lipgloss.NewStyle().
			Foreground(ColorDim).
			Render("Waiting for Codex session..."))

		// Show usage even while waiting (WHAM API doesn't need a session).
		if m.State.HasRateLimits {
			parts = append(parts, "")
			parts = append(parts, RenderRateLimit(m.State, innerWidth))
		}

		body := lipgloss.JoinVertical(lipgloss.Left, parts...)
		return OuterStyle.Width(m.Width - 4).Height(m.Height - 4).Render(body)
	}

	// Compute available inner width.
	innerWidth := m.Width - 8
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Available inner height: total height minus outer border (2) and padding (2).
	innerHeight := m.Height - 4

	// Compact mode only when very tight (< 12 usable lines).
	compact := innerHeight < 12

	var sections []string

	// Header (always shown — 1 line)
	sections = append(sections, RenderHeader(m.State, innerWidth))

	if compact {
		// Compact: context in single line (no card border)
		sections = append(sections, RenderContextCompact(m.State, innerWidth))

		// Compact: skills right after context
		if sk := RenderSkillsCompact(m.State, innerWidth); sk != "" {
			sections = append(sections, sk)
		}

		// Compact: tokens + session merged into 2 lines
		sections = append(sections, RenderTokensCompact(m.State, innerWidth))
		sections = append(sections, RenderSessionCompact(m.State, m.GitStatus, innerWidth))

		// Compact: usage in 2 lines (no card border)
		if m.Config.Display.ShowRateLimit {
			if rl := RenderRateLimitCompact(m.State, innerWidth); rl != "" {
				sections = append(sections, rl)
			}
		}

		// Compact: activity in 1 line
		if m.Config.Display.ShowActivity {
			if act := RenderActivityCompact(m.State, innerWidth); act != "" {
				sections = append(sections, act)
			}
		}
	} else {
		// Normal: full card layout
		sections = append(sections, RenderContext(m.State, innerWidth))

		// Skills right after context
		if sk := RenderSkillsCompact(m.State, innerWidth); sk != "" {
			sections = append(sections, sk)
		}

		if m.Width >= 80 {
			halfWidth := innerWidth / 2

			tokensCard := RenderTokens(m.State, halfWidth)
			sessionCard := RenderSession(m.State, m.GitStatus, halfWidth)

			tLines := strings.Count(tokensCard, "\n") + 1
			sLines := strings.Count(sessionCard, "\n") + 1

			maxLines := tLines
			if sLines > maxLines {
				maxLines = sLines
			}
			minContent := maxLines - 2
			tokensCard = RenderTokens(m.State, halfWidth, minContent)
			sessionCard = RenderSession(m.State, m.GitStatus, halfWidth, minContent)

			row := lipgloss.JoinHorizontal(lipgloss.Top, tokensCard, sessionCard)
			sections = append(sections, row)
		} else {
			sections = append(sections, RenderTokens(m.State, innerWidth))
			sections = append(sections, RenderSession(m.State, m.GitStatus, innerWidth))
		}

		// For remaining sections, check available vertical space and fall
		// back to compact rendering when the full card would be clipped.
		usedHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, sections...))
		remaining := innerHeight - usedHeight

		if m.Config.Display.ShowRateLimit {
			if full := RenderRateLimit(m.State, innerWidth); full != "" {
				if lipgloss.Height(full) <= remaining {
					sections = append(sections, full)
				} else if c := RenderRateLimitCompact(m.State, innerWidth); c != "" {
					sections = append(sections, c)
				}
				usedHeight = lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, sections...))
				remaining = innerHeight - usedHeight
			}
		}

		if m.Config.Display.ShowActivity {
			if full := RenderActivity(m.State, innerWidth); full != "" {
				if lipgloss.Height(full) <= remaining {
					sections = append(sections, full)
				} else if c := RenderActivityCompact(m.State, innerWidth); c != "" {
					sections = append(sections, c)
				}
			}
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)
	rendered := OuterStyle.Width(m.Width - 4).Height(m.Height - 4).Render(body)

	// Clip to terminal height so the top (header) is always visible.
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.Height {
		lines = lines[:m.Height]
	}
	return strings.Join(lines, "\n")
}
