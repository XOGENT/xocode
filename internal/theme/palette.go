// Package theme is the single source of visual truth for xocode. No color
// literals or lipgloss styles should live anywhere else in the codebase.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds the raw color tokens. Ghostty-inspired, dark-first, with the
// XOGENT violet accent. AdaptiveColor keeps the UI legible on light terminals.
var (
	ColorBG       = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0D0D0F"}
	ColorSurface  = lipgloss.AdaptiveColor{Light: "#F2F2F5", Dark: "#16161A"}
	ColorSurface2 = lipgloss.AdaptiveColor{Light: "#E7E7EC", Dark: "#1E1E24"}
	ColorBorder   = lipgloss.AdaptiveColor{Light: "#D7D7DE", Dark: "#2A2A32"}

	ColorFG    = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E6E6E6"}
	ColorMuted = lipgloss.AdaptiveColor{Light: "#6B6B76", Dark: "#7A7A85"}
	ColorFaint = lipgloss.AdaptiveColor{Light: "#9A9AA3", Dark: "#4E4E58"}

	ColorAccent    = lipgloss.AdaptiveColor{Light: "#6D46FF", Dark: "#7C5CFF"} // XOGENT violet
	ColorAccentDim = lipgloss.AdaptiveColor{Light: "#B9A8FF", Dark: "#4A3A99"}
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	ColorWarn      = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}
	ColorDanger    = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}

	// ColorOnAccent is text drawn on top of an accent fill.
	ColorOnAccent = lipgloss.Color("#FFFFFF")
)
