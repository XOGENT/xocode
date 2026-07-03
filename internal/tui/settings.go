package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/xogent/xocode/internal/config"
)

// settingsField describes one editable option and its preset choices.
type settingsField struct {
	label   string
	choices []string
	get     func(*config.Config) string
	set     func(*config.Config, string)
}

var settingsFields = []settingsField{
	{
		label:   "Claude model",
		choices: []string{"opus", "sonnet", "haiku"},
		get:     func(c *config.Config) string { return c.ClaudeModel },
		set:     func(c *config.Config, v string) { c.ClaudeModel = v },
	},
	{
		label:   "Claude effort",
		choices: []string{"low", "medium", "high", "xhigh"},
		get:     func(c *config.Config) string { return c.ClaudeEffort },
		set:     func(c *config.Config, v string) { c.ClaudeEffort = v },
	},
	{
		label:   "Cursor model",
		choices: []string{"composer-2.5", "composer-1", "auto"},
		get:     func(c *config.Config) string { return c.CursorModel },
		set:     func(c *config.Config, v string) { c.CursorModel = v },
	},
}

// updateSettings handles the settings editor.
func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "esc", "q":
		m.state = m.settingsReturnState()
		m.focusForState()
		return m, nil
	case "up", "k":
		if m.setCursor > 0 {
			m.setCursor--
		}
	case "down", "j":
		if m.setCursor < len(settingsFields)-1 {
			m.setCursor++
		}
	case "left", "h":
		m.cycleSetting(-1)
	case "right", "l":
		m.cycleSetting(1)
	case "enter":
		m.cfg.ClaudeModel = m.setDraft.ClaudeModel
		m.cfg.ClaudeEffort = m.setDraft.ClaudeEffort
		m.cfg.CursorModel = m.setDraft.CursorModel
		_ = config.Save(m.cfg) // best-effort; UI still applies the change
		m.state = m.settingsReturnState()
		m.focusForState()
		return m, nil
	}
	return m, nil
}

func (m Model) settingsReturnState() State {
	if m.prev == StatePlanning || m.prev == StateReview || m.prev == StateSummary {
		return m.prev
	}
	return StateInput
}

func (m *Model) focusForState() {
	if m.state == StateInput || (m.state == StatePlanning && !m.running) {
		m.composer.Focus()
	}
}

// cycleSetting moves the current field to the next/previous preset choice.
func (m *Model) cycleSetting(dir int) {
	f := settingsFields[m.setCursor]
	cur := f.get(&m.setDraft)
	idx := 0
	for i, c := range f.choices {
		if c == cur {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(f.choices)) % len(f.choices)
	f.set(&m.setDraft, f.choices[idx])
}

func (m Model) viewSettings() string {
	title := m.th.Title.Render("Settings")
	lines := []string{title, "", m.th.Muted.Render("Saved to " + config.StateDir()), ""}
	for i, f := range settingsFields {
		cur := f.get(&m.setDraft)
		name := m.th.Body.Render(pad(f.label, 16))
		var val string
		if i == m.setCursor {
			name = m.th.Accent.Render(pad("› "+f.label, 16))
			val = m.th.ListSelected.Render(" " + cur + " ")
		} else {
			val = m.th.Badge.Render(cur)
		}
		lines = append(lines, name+"  "+m.th.Faint.Render("‹ ")+val+m.th.Faint.Render(" ›"))
	}
	box := m.th.PanelActive.Width(m.width - 4).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return m.fill(box)
}
