// Package tui implements the xocode terminal UI as a Bubble Tea state machine.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the interactive TUI. It blocks until the user quits.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
