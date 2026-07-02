package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/xogent/xocode/internal/git"
	"github.com/xogent/xocode/internal/stream"
	"github.com/xogent/xocode/internal/theme"
)

// View renders the current state.
func (m Model) View() string {
	if !m.ready {
		return ""
	}
	var body string
	switch m.state {
	case StatePreflight:
		body = m.viewPreflight()
	case StateInput:
		body = m.viewInput()
	case StatePlanning, StateBuilding:
		body = m.viewStreaming()
	case StateReview:
		body = m.viewReview()
	case StateConfirmInit:
		body = m.viewConfirmInit()
	case StateSummary:
		body = m.viewSummary()
	case StateError:
		body = m.viewError()
	default:
		body = ""
	}
	return strings.Join([]string{m.renderHeader(), body, m.renderFooter()}, "\n")
}

func (m Model) viewPreflight() string {
	title := m.th.Title.Render("Checking prerequisites")
	lines := []string{title, ""}
	if m.checking && len(m.checks) == 0 {
		lines = append(lines, m.spinner.View()+" "+m.th.Muted.Render("Verifying Claude Code and Cursor…"))
	}
	for _, r := range m.checks {
		var mark string
		var detail string
		switch {
		case r.OK():
			mark = m.th.Success.Render("✓")
			detail = m.th.Muted.Render("installed, logged in")
			if r.Detail != "" {
				detail = m.th.Muted.Render(r.Detail)
			}
		case r.Installed && !r.LoggedIn:
			mark = m.th.Warn.Render("•")
			detail = m.th.Warn.Render(r.Fix)
		default:
			mark = m.th.Danger.Render("✗")
			detail = m.th.Danger.Render(r.Fix)
		}
		lines = append(lines, mark+" "+m.th.Body.Render(pad(r.Name, 13))+" "+detail)
	}
	if m.checking && len(m.checks) > 0 {
		lines = append(lines, "", m.spinner.View()+" "+m.th.Muted.Render("Working…"))
	}
	box := m.th.PanelActive.Width(m.width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}

func pad(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func (m Model) viewInput() string {
	prompt := m.th.Title.Render("What should we build?")
	hint := m.th.Muted.Render("Opus 4.8 will plan it; Composer 2.5 will build it.")
	box := m.th.PanelActive.Width(m.width - 2).Render(m.task.View())
	content := lipgloss.JoinVertical(lipgloss.Left, prompt, hint, "", box)
	return m.fill(content)
}

func (m Model) viewStreaming() string {
	status := m.spinner.View() + " " + m.th.Accent.Render(m.state.label()+"…")
	if m.activeTool != "" {
		status += m.th.Muted.Render("  ·  ") + m.th.ToolBadge.Render(m.activeTool)
	}
	// The viewport already occupies the content area; drop the status onto its
	// first line by reserving one row.
	return status + "\n" + m.vp.View()
}

func (m Model) viewReview() string {
	return m.vp.View()
}

func (m Model) viewConfirmInit() string {
	title := m.th.Title.Render("This directory isn't a git repository")
	msg := m.th.Body.Render(
		"xocode builds in an isolated git worktree so your files stay untouched\n" +
			"until you merge. That requires a git repo here.")
	q := m.th.Accent.Render("Initialize a git repository in this directory now? (y/n)")
	box := m.th.PanelActive.Width(m.width - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", q))
	return m.fill(box)
}

func (m Model) viewSummary() string {
	title := m.th.Success.Render("✓ Build complete")
	lines := []string{title, ""}
	if s := strings.TrimSpace(m.buildSummary); s != "" {
		lines = append(lines, m.th.Body.Render(truncateLines(s, 8)), "")
	}
	lines = append(lines, m.th.Subtitle.Render("Worktree"))
	lines = append(lines, m.th.Muted.Render(m.worktreePath), "")
	lines = append(lines, m.th.Subtitle.Render("Next steps"))
	for _, g := range git.MergeGuidance(m.projectDir, m.worktreePath) {
		lines = append(lines, m.th.Body.Render("  "+g))
	}
	box := m.th.Panel.Width(m.width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}

// truncateLines keeps at most n lines, appending an ellipsis if truncated.
func truncateLines(s string, n int) string {
	ls := strings.Split(s, "\n")
	if len(ls) <= n {
		return s
	}
	return strings.Join(ls[:n], "\n") + "\n…"
}

func (m Model) viewError() string {
	title := m.th.Danger.Render("✗ Something went wrong")
	msg := ""
	if m.err != nil {
		msg = m.err.Error()
	}
	box := m.th.Panel.
		BorderForeground(theme.ColorDanger).
		Width(m.width - 2).
		Render(m.th.Body.Render(msg))
	return m.fill(lipgloss.JoinVertical(lipgloss.Left, title, "", box))
}

// fill vertically centers content within the body area.
func (m Model) fill(content string) string {
	h := m.vp.Height
	if h < 1 {
		h = 1
	}
	return lipgloss.Place(m.width, h, lipgloss.Left, lipgloss.Top, content)
}

func (m Model) renderHeader() string {
	brand := m.th.Accent.Render(m.th.Wordmark)
	crumb := m.th.Subtitle.Render("  ›  " + m.state.label())
	return m.th.Header.Width(m.width).Render(brand + crumb)
}

func (m Model) renderFooter() string {
	var hints []string
	switch m.state {
	case StatePreflight:
		if m.checking {
			hints = []string{m.keyHint("ctrl+c", "quit")}
		} else {
			hints = []string{m.keyHint("i", "install & log in"), m.keyHint("r", "re-check"), m.keyHint("q", "quit")}
		}
	case StateInput:
		hints = []string{m.keyHint("ctrl+d", "submit"), m.keyHint("ctrl+c", "quit")}
	case StatePlanning, StateBuilding:
		hints = []string{m.keyHint("esc", "cancel"), m.keyHint("ctrl+c", "quit")}
	case StateReview:
		hints = []string{m.keyHint("enter", "approve & build"), m.keyHint("e", "edit"), m.keyHint("↑/↓", "scroll"), m.keyHint("q", "quit")}
	case StateConfirmInit:
		hints = []string{m.keyHint("y", "init & build"), m.keyHint("n", "back")}
	case StateSummary:
		hints = []string{m.keyHint("n", "new task"), m.keyHint("q", "quit")}
	case StateError:
		hints = []string{m.keyHint("enter", "back"), m.keyHint("q", "quit")}
	}
	return m.th.Footer.Width(m.width).Render(strings.Join(hints, m.th.Muted.Render("   ")))
}

func (m Model) keyHint(k, desc string) string {
	return m.th.KeyHint.Render(k) + " " + m.th.KeyDesc.Render(desc)
}

// refreshStreamViewport re-renders the streamed log into the viewport and pins
// it to the bottom.
func (m *Model) refreshStreamViewport() {
	if !m.ready {
		return
	}
	width := m.vp.Width
	var b strings.Builder
	for i, e := range m.log {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderLogEntry(e, width))
	}
	m.vp.SetContent(b.String())
	m.vp.GotoBottom()
}

func (m Model) renderLogEntry(e logEntry, width int) string {
	wrap := lipgloss.NewStyle().Width(width)
	switch e.kind {
	case stream.KindToolUse:
		line := m.th.ToolBadge.Render("⚙ " + e.tool)
		if e.info != "" {
			line += m.th.Muted.Render(" " + e.info)
		}
		if e.text != "" {
			line = wrap.Render(m.th.Body.Render(e.text)) + "\n" + line
		}
		return line
	default:
		return wrap.Render(m.th.Body.Render(e.text))
	}
}

// setReviewContent renders the plan markdown into the review viewport.
func (m *Model) setReviewContent(text string) {
	if !m.ready {
		return
	}
	m.vp.SetContent(renderMarkdown(text, m.vp.Width))
	m.vp.GotoTop()
}

// renderMarkdown renders plan markdown with glamour, falling back to plain
// wrapped text if the renderer can't be built.
func renderMarkdown(text string, width int) string {
	if width < 10 {
		width = 10
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(text)
	}
	out, err := r.Render(text)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(text)
	}
	return out
}

// glamourStyle picks a glamour theme that matches the terminal background.
func glamourStyle() string {
	if lipgloss.HasDarkBackground() {
		return "dark"
	}
	return "light"
}
