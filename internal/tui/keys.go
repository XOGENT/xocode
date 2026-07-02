package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Submit  key.Binding
	Approve key.Binding
	Edit    key.Binding
	Cancel  key.Binding
	Quit    key.Binding
	Help    key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Submit: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "submit"),
		),
		Approve: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "approve & build"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit plan"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
