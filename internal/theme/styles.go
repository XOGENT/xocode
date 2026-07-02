package theme

import "github.com/charmbracelet/lipgloss"

// Theme is the prebuilt set of styles used across the TUI. Build it once with
// New() and pass it into the model.
type Theme struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Muted    lipgloss.Style

	Accent  lipgloss.Style
	Success lipgloss.Style
	Warn    lipgloss.Style
	Danger  lipgloss.Style

	Panel       lipgloss.Style
	PanelActive lipgloss.Style

	Header    lipgloss.Style
	Footer    lipgloss.Style
	KeyHint   lipgloss.Style
	KeyDesc   lipgloss.Style
	ToolBadge lipgloss.Style
	Spinner   lipgloss.Style

	// Wordmark is the XOGENT brand shown in the header.
	Wordmark string
}

// New builds the Theme from the palette.
func New() Theme {
	return Theme{
		Title:    lipgloss.NewStyle().Foreground(ColorFG).Bold(true),
		Subtitle: lipgloss.NewStyle().Foreground(ColorMuted),
		Body:     lipgloss.NewStyle().Foreground(ColorFG),
		Muted:    lipgloss.NewStyle().Foreground(ColorMuted),

		Accent:  lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),
		Success: lipgloss.NewStyle().Foreground(ColorSuccess),
		Warn:    lipgloss.NewStyle().Foreground(ColorWarn),
		Danger:  lipgloss.NewStyle().Foreground(ColorDanger),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1),
		PanelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1),

		Header: lipgloss.NewStyle().
			Foreground(ColorFG).
			Bold(true).
			Padding(0, 1),
		Footer: lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1),
		KeyHint:   lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),
		KeyDesc:   lipgloss.NewStyle().Foreground(ColorMuted),
		ToolBadge: lipgloss.NewStyle().Foreground(ColorAccentDim).Bold(true),
		Spinner:   lipgloss.NewStyle().Foreground(ColorAccent),

		Wordmark: "XOCODE",
	}
}
