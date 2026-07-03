package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// matchesKey reports whether a key message activates a binding.
func matchesKey(k tea.KeyMsg, b key.Binding) bool {
	return key.Matches(k, b)
}

// keyMap holds every binding the TUI uses. It also satisfies help.KeyMap so the
// `?` overlay renders from a single source of truth.
type keyMap struct {
	Send     key.Binding
	Newline  key.Binding
	Approve  key.Binding
	Edit     key.Binding
	Refine   key.Binding
	Discard  key.Binding
	Cancel   key.Binding
	History  key.Binding
	Settings key.Binding
	Scroll   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
		),
		Newline: key.NewBinding(
			key.WithKeys("alt+enter", "shift+enter", "ctrl+j"),
			key.WithHelp("alt+enter", "newline"),
		),
		Approve: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "approve & build"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Refine: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refine"),
		),
		Discard: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "discard"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		History: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "history"),
		),
		Settings: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "settings"),
		),
		Scroll: key.NewBinding(
			key.WithKeys("up", "down", "pgup", "pgdown"),
			key.WithHelp("↑/↓", "scroll"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// ShortHelp / FullHelp implement help.KeyMap for the `?` overlay.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.Approve, k.Edit, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.Newline},
		{k.Approve, k.Edit, k.Refine, k.Discard},
		{k.History, k.Settings, k.Scroll},
		{k.Cancel, k.Help, k.Quit},
	}
}
