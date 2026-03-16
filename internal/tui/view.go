package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// NewViewport creates a viewport with mouse wheel scrolling enabled.
func NewViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3
	return vp
}

// View renders the full TUI layout.
func (m Model) View() string {
	if !m.vpReady {
		// Viewport not initialized yet — show minimal placeholder.
		return ""
	}

	vpView := m.Viewport.View()

	// Show a subtle scroll indicator when content is taller than the viewport.
	scrollHint := ""
	if m.Viewport.TotalLineCount() > m.Viewport.Height {
		pct := m.Viewport.ScrollPercent() * 100
		scrollHint = lipgloss.NewStyle().
			Foreground(ColorDim).
			Render(fmt.Sprintf(" ↕ %.0f%%", pct))
	}

	// Build the outer frame. Use Width only (no fixed Height) so the border
	// wraps snugly around the viewport.
	outerWidth := m.Width - 4
	if outerWidth < 14 {
		outerWidth = 14
	}

	// Combine viewport + scroll hint in a fixed-height frame.
	inner := vpView
	if scrollHint != "" {
		// Place scroll hint on the last visible line, right-aligned.
		lines := strings.Split(inner, "\n")
		if len(lines) > 0 {
			lastIdx := len(lines) - 1
			lines[lastIdx] = lines[lastIdx] + scrollHint
		}
		inner = strings.Join(lines, "\n")
	}

	rendered := OuterStyle.Width(outerWidth).Height(m.Height - 4).Render(inner)

	// Clip to terminal height as a safety net.
	outLines := strings.Split(rendered, "\n")
	if len(outLines) > m.Height {
		outLines = outLines[:m.Height]
	}
	return strings.Join(outLines, "\n")
}

// renderBody generates the inner content string for the viewport.
func (m *Model) renderBody() string {
	// Inner width available for content (inside outer border + padding).
	innerWidth := m.Width - 8
	if innerWidth < 20 {
		innerWidth = 20
	}

	if m.Waiting {
		var parts []string
		parts = append(parts, lipgloss.NewStyle().
			Foreground(ColorDim).
			Render("Waiting for Codex session..."))

		if m.State.HasRateLimits {
			parts = append(parts, "")
			parts = append(parts, RenderRateLimit(m.State, innerWidth))
		}
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}

	// Available inner height (rough estimate for compact mode decision).
	innerHeight := m.Height - 4
	compact := innerHeight < 12

	var sections []string

	// Header (always shown — 1 line)
	sections = append(sections, RenderHeader(m.State, innerWidth))

	if compact {
		sections = append(sections, RenderContextCompact(m.State, innerWidth))

		if sk := RenderSkillsCompact(m.State, innerWidth); sk != "" {
			sections = append(sections, sk)
		}

		sections = append(sections, RenderTokensCompact(m.State, innerWidth))
		sections = append(sections, RenderSessionCompact(m.State, m.GitStatus, innerWidth))

		if m.Config.Display.ShowRateLimit {
			if rl := RenderRateLimitCompact(m.State, innerWidth); rl != "" {
				sections = append(sections, rl)
			}
		}

		if m.Config.Display.ShowActivity {
			if act := RenderActivityCompact(m.State, innerWidth); act != "" {
				sections = append(sections, act)
			}
		}
	} else {
		// Normal: full card layout — render ALL sections without height
		// clipping. The viewport handles scrolling if content overflows.
		sections = append(sections, RenderContext(m.State, innerWidth))

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

		// Always render remaining sections — viewport scrolls if needed.
		if m.Config.Display.ShowRateLimit {
			if rl := RenderRateLimit(m.State, innerWidth); rl != "" {
				sections = append(sections, rl)
			}
		}

		if m.Config.Display.ShowActivity {
			if act := RenderActivity(m.State, innerWidth); act != "" {
				sections = append(sections, act)
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
