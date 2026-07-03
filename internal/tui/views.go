package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/xogent/xocode/internal/git"
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
	case StateInput, StatePlanning:
		body = m.viewConversation()
	case StateReview:
		body = m.viewReview()
	case StateBuilding:
		body = m.viewBuilding()
	case StateConfirmInit:
		body = m.viewConfirmInit()
	case StateSummary:
		body = m.viewSummary()
	case StateHistory:
		body = m.viewHistory()
	case StateSettings:
		body = m.viewSettings()
	case StateError:
		body = m.viewError()
	}
	screen := strings.Join([]string{m.renderHeader(), body, m.renderFooter()}, "\n")
	if m.showHelp {
		return m.overlay(screen, m.renderHelp())
	}
	return screen
}

// bodyHeight is the number of rows available between header and footer.
func (m Model) bodyHeight() int {
	h := m.height - lipgloss.Height(m.renderHeader()) - lipgloss.Height(m.renderFooter())
	if h < 1 {
		h = 1
	}
	return h
}

// ---------- header / footer ----------

func (m Model) renderHeader() string {
	brand := m.th.Accent.Render(m.th.Wordmark)
	stepper := m.renderStepper()
	left := brand + "  " + stepper
	right := ""
	if m.state == StateInput || m.state == StatePlanning || m.state == StateReview {
		right = m.th.Badge.Render(m.cfg.ClaudeModel + " · " + m.cfg.ClaudeEffort)
	} else if m.state == StateBuilding {
		right = m.th.Badge.Render(m.cfg.CursorModel)
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return m.th.Header.Render(line)
}

// renderStepper draws the Task › Plan › Review › Build › Done progress line.
func (m Model) renderStepper() string {
	cur := m.state.stepIndex()
	if cur < 0 {
		// Off-flow screens (history/settings/error): just name the screen.
		return m.th.StepSep.Render("· ") + m.th.Subtitle.Render(m.state.label())
	}
	var parts []string
	for i, s := range steps {
		switch {
		case i < cur:
			parts = append(parts, m.th.StepDone.Render("✓ "+s.label))
		case i == cur:
			parts = append(parts, m.th.StepActive.Render(s.label))
		default:
			parts = append(parts, m.th.StepUpcoming.Render(s.label))
		}
	}
	return strings.Join(parts, m.th.StepSep.Render(" › "))
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
		hints = []string{m.keyHint("enter", "send"), m.keyHint("ctrl+r", "history"), m.keyHint("ctrl+s", "settings"), m.keyHint("ctrl+c", "quit")}
	case StatePlanning:
		if m.running {
			hints = []string{m.keyHint("esc", "stop"), m.keyHint("ctrl+c", "quit")}
		} else {
			hints = []string{m.keyHint("enter", "send"), m.keyHint("alt+enter", "newline"), m.keyHint("esc", "reset"), m.keyHint("ctrl+c", "quit")}
		}
	case StateReview:
		hints = []string{m.keyHint("enter", "approve & build"), m.keyHint("e", "edit"), m.keyHint("r", "refine"), m.keyHint("d", "discard"), m.keyHint("?", "help")}
	case StateBuilding:
		hints = []string{m.keyHint("esc", "cancel"), m.keyHint("ctrl+c", "quit")}
	case StateConfirmInit:
		hints = []string{m.keyHint("y", "init & build"), m.keyHint("n", "back")}
	case StateSummary:
		hints = []string{m.keyHint("n", "new task"), m.keyHint("q", "quit")}
	case StateHistory:
		hints = []string{m.keyHint("↑/↓", "select"), m.keyHint("enter", "open"), m.keyHint("esc", "back")}
	case StateSettings:
		hints = []string{m.keyHint("↑/↓", "field"), m.keyHint("←/→", "change"), m.keyHint("enter", "save"), m.keyHint("esc", "cancel")}
	case StateError:
		hints = []string{m.keyHint("enter", "back"), m.keyHint("q", "quit")}
	}
	return m.th.Footer.Render(strings.Join(hints, m.th.Faint.Render("   ")))
}

func (m Model) keyHint(k, desc string) string {
	return m.th.KeyHint.Render(k) + " " + m.th.KeyDesc.Render(desc)
}

// ---------- status bar ----------

func (m Model) renderStatusBar(verb string) string {
	var left string
	if m.running {
		left = m.spinner.View() + " " + m.th.Accent.Render(verb)
		if m.activeTool != "" {
			left += "  " + m.th.ToolBadge.Render("⚙ "+m.activeTool)
		}
	} else {
		left = m.th.Muted.Render(verb)
	}

	var segs []string
	if m.running || m.elapsed > 0 {
		segs = append(segs, fmt.Sprintf("%02d:%02d", int(m.elapsed.Minutes()), int(m.elapsed.Seconds())%60))
	}
	if !m.usage.Empty() {
		segs = append(segs, fmt.Sprintf("↑%s ↓%s", humanTokens(m.usage.InputTokens), humanTokens(m.usage.OutputTokens)))
		if m.usage.CostUSD > 0 {
			segs = append(segs, fmt.Sprintf("$%.3f", m.usage.CostUSD))
		}
	}
	right := m.th.StatusBar.Render(strings.Join(segs, "  ·  "))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func humanTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// ---------- preflight ----------

func (m Model) viewPreflight() string {
	title := m.th.Title.Render("Checking prerequisites")
	lines := []string{title, ""}
	if m.checking && len(m.checks) == 0 {
		lines = append(lines, m.spinner.View()+" "+m.th.Muted.Render("Verifying Claude Code and Cursor…"))
	}
	for _, r := range m.checks {
		var mark, detail string
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
	box := m.th.PanelActive.Width(m.width - 4).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}

// ---------- conversation (input + planning) ----------

func (m Model) viewConversation() string {
	verb := "Ready"
	switch {
	case m.running:
		verb = "Planning…"
	case m.state == StateInput:
		verb = "What should we build?"
	case len(m.chat) > 0:
		verb = "Your turn"
	}
	status := m.renderStatusBar(verb)

	m.chatVP.Height = m.bodyHeight() - 6
	if m.chatVP.Height < 1 {
		m.chatVP.Height = 1
	}
	transcript := m.chatVP.View()

	composerStyle := m.th.ComposerBlurred
	if m.composer.Focused() {
		composerStyle = m.th.ComposerFocused
	}
	composer := composerStyle.Width(m.width - 4).Render(m.composer.View())

	return strings.Join([]string{status, transcript, composer}, "\n")
}

// refreshChatViewport re-renders the planning transcript, pinned to the bottom.
func (m *Model) refreshChatViewport() {
	if !m.ready {
		return
	}
	m.chatVP.SetContent(m.renderTranscript(m.chat))
	m.chatVP.GotoBottom()
}

func (m *Model) refreshBuildViewport() {
	if !m.ready {
		return
	}
	m.chatVP.SetContent(m.renderTranscript(m.buildLog))
	m.chatVP.GotoBottom()
}

// renderTranscript renders a slice of chat messages, caching per-message output.
func (m *Model) renderTranscript(msgs []chatMsg) string {
	width := m.chatVP.Width
	if len(msgs) == 0 {
		return m.welcomeHero(width)
	}
	var b strings.Builder
	for i := range msgs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if msgs[i].rendered == "" {
			msgs[i].rendered = m.renderChatMsg(msgs[i], width)
		}
		b.WriteString(msgs[i].rendered)
	}
	return b.String()
}

func (m Model) welcomeHero(width int) string {
	lines := []string{
		m.th.Accent.Render("▲ xocode"),
		"",
		m.th.Body.Render("Plan with Opus 4.8. Build with Composer 2.5."),
		m.th.Muted.Render("Describe a change below and press enter. Say hi to chat first."),
	}
	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderChatMsg(e chatMsg, width int) string {
	switch e.role {
	case roleUser:
		body := lipgloss.NewStyle().Width(width - 3).Render(m.th.Body.Render(e.text))
		content := m.th.ChatUserLabel.Render("you") + "\n" + body
		return m.th.ChatUser.Render(content)
	case roleTool:
		line := m.th.ToolBadge.Render(toolGlyph(e.tool) + " " + e.tool)
		if e.info != "" {
			line += m.th.Faint.Render("  " + truncateMid(e.info, width-lipgloss.Width(line)-4))
		}
		return line
	default:
		label := m.th.ChatAssistLbl.Render(m.cfg.ClaudeModel)
		return label + "\n" + renderMarkdown(e.text, width)
	}
}

// toolGlyph maps common tool names to an ASCII-safe glyph.
func toolGlyph(name string) string {
	switch name {
	case "Read", "NotebookRead":
		return "▸"
	case "Edit", "Write", "MultiEdit", "NotebookEdit", "apply_patch", "str_replace_editor", "Create":
		return "✎"
	case "Bash":
		return "❯"
	case "Grep", "Glob", "Search":
		return "⌕"
	case "WebFetch", "WebSearch":
		return "◈"
	case "Task":
		return "⧉"
	case "cancelled":
		return "■"
	default:
		return "⚙"
	}
}

// ---------- review ----------

func (m Model) viewReview() string {
	pct := int(m.reviewVP.ScrollPercent() * 100)
	title := m.th.Title.Render("Implementation plan")
	scroll := m.th.ScrollHl.Render(fmt.Sprintf("%d%%", pct))
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(scroll) - 2
	if gap < 1 {
		gap = 1
	}
	header := " " + title + strings.Repeat(" ", gap) + scroll + " "
	m.reviewVP.Height = m.bodyHeight() - 1
	if m.reviewVP.Height < 1 {
		m.reviewVP.Height = 1
	}
	return header + "\n" + m.reviewVP.View()
}

// ---------- building ----------

func (m Model) viewBuilding() string {
	status := m.renderStatusBar("Building…")
	files := m.renderBuildFiles()
	m.chatVP.Height = m.bodyHeight() - 2
	if m.chatVP.Height < 1 {
		m.chatVP.Height = 1
	}
	return strings.Join([]string{status, files, m.chatVP.View()}, "\n")
}

func (m Model) renderBuildFiles() string {
	if len(m.buildFiles) == 0 {
		return m.th.Faint.Render(" no files changed yet")
	}
	names := make([]string, 0, len(m.buildFiles))
	for _, f := range m.buildFiles {
		names = append(names, baseName(f))
	}
	label := m.th.Subtitle.Render(fmt.Sprintf(" Files changed (%d): ", len(names)))
	list := m.th.Body.Render(truncateMid(strings.Join(names, ", "), m.width-lipgloss.Width(label)-2))
	return label + list
}

// ---------- confirm init ----------

func (m Model) viewConfirmInit() string {
	title := m.th.Title.Render("This directory isn't a git repository")
	msg := m.th.Body.Render(
		"xocode builds in an isolated git worktree so your files stay untouched\n" +
			"until you merge. That requires a git repo here.")
	q := m.th.Accent.Render("Initialize a git repository in this directory now? (y/n)")
	box := m.th.PanelActive.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", q))
	return m.fill(box)
}

// ---------- summary ----------

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
	box := m.th.Panel.Width(m.width - 4).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}

// ---------- history ----------

func (m Model) viewHistory() string {
	title := m.th.Title.Render("Saved plans")
	lines := []string{title, ""}
	if len(m.history) == 0 {
		lines = append(lines, m.th.Muted.Render("No saved plans yet."))
	}
	for i, p := range m.history {
		label := fmt.Sprintf("%s  %s", p.CreatedAt.Format("2006-01-02 15:04"), firstLine(p.Task))
		style := m.th.ListItem
		if i == m.histCursor {
			style = m.th.ListSelected
		}
		lines = append(lines, style.Width(m.width-8).Render(truncateMid(label, m.width-10)))
	}
	box := m.th.Panel.Width(m.width - 4).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}

// ---------- error ----------

func (m Model) viewError() string {
	title := m.th.Danger.Render("✗ Something went wrong")
	msg := ""
	if m.err != nil {
		msg = m.err.Error()
	}
	box := m.th.Panel.
		BorderForeground(theme.ColorDanger).
		Width(m.width - 4).
		Render(m.th.Body.Render(msg))
	return m.fill(lipgloss.JoinVertical(lipgloss.Left, title, "", box))
}

// ---------- help overlay ----------

func (m Model) renderHelp() string {
	title := m.th.Title.Render("Keyboard shortcuts")
	body := m.help.View(m.keys)
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", m.th.Muted.Render("press any key to close"))
	return m.th.Overlay.Render(content)
}

// overlay centers a modal over a dimmed screen.
func (m Model) overlay(screen, modal string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "))
}

// ---------- shared helpers ----------

func (m Model) fill(content string) string {
	h := m.bodyHeight()
	return lipgloss.Place(m.width, h, lipgloss.Left, lipgloss.Top, content)
}

// setReviewContent renders the plan markdown into the review viewport.
func (m *Model) setReviewContent(text string) {
	if !m.ready {
		return
	}
	m.reviewVP.SetContent(renderMarkdown(text, m.reviewVP.Width))
	m.reviewVP.GotoTop()
}

// renderMarkdown renders markdown with glamour, falling back to wrapped text.
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
	return strings.TrimRight(out, "\n")
}

func glamourStyle() string {
	if lipgloss.HasDarkBackground() {
		return "dark"
	}
	return "light"
}

func pad(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func truncateLines(s string, n int) string {
	ls := strings.Split(s, "\n")
	if len(ls) <= n {
		return s
	}
	return strings.Join(ls[:n], "\n") + "\n…"
}

// truncateMid trims a string to width, adding an ellipsis if needed.
func truncateMid(s string, width int) string {
	if width < 1 {
		width = 1
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + "…"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func baseName(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
