// Package theme is the single source of visual truth for xocode. No color
// literals or lipgloss styles should live anywhere else in the codebase.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds the raw color tokens. Ghostty-inspired dark-minimal with a
// XOGENT violet accent. AdaptiveColor lets the UI degrade gracefully on light
// terminals.
var (
	ColorBG        = lipgloss.Color("#0D0D0F")
	ColorSurface   = lipgloss.Color("#16161A")
	ColorFG        = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E6E6E6"}
	ColorMuted     = lipgloss.AdaptiveColor{Light: "#6B6B76", Dark: "#7A7A85"}
	ColorAccent    = lipgloss.Color("#7C5CFF") // XOGENT violet
	ColorAccentDim = lipgloss.Color("#4A3A99")
	ColorSuccess   = lipgloss.Color("#3FB950")
	ColorWarn      = lipgloss.Color("#D29922")
	ColorDanger    = lipgloss.Color("#F85149")
)
