package theme

import "github.com/charmbracelet/lipgloss"

// Theme is the prebuilt set of styles used across the TUI. Build it once with
// New() and pass it into the model.
type Theme struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Muted    lipgloss.Style
	Faint    lipgloss.Style

	Accent  lipgloss.Style
	Success lipgloss.Style
	Warn    lipgloss.Style
	Danger  lipgloss.Style

	Panel       lipgloss.Style
	PanelActive lipgloss.Style
	Overlay     lipgloss.Style

	Header  lipgloss.Style
	Footer  lipgloss.Style
	KeyHint lipgloss.Style
	KeyDesc lipgloss.Style
	Spinner lipgloss.Style

	// Stepper
	StepDone     lipgloss.Style
	StepActive   lipgloss.Style
	StepUpcoming lipgloss.Style
	StepSep      lipgloss.Style

	// Chat transcript
	ChatUser      lipgloss.Style
	ChatUserLabel lipgloss.Style
	ChatAssistLbl lipgloss.Style
	ToolBadge     lipgloss.Style

	// Status bar & badges
	StatusBar lipgloss.Style
	Badge     lipgloss.Style
	CostBadge lipgloss.Style
	ScrollHl  lipgloss.Style

	// Composer
	ComposerFocused lipgloss.Style
	ComposerBlurred lipgloss.Style

	// Lists (history / settings)
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style

	// Action bar
	ActionKey lipgloss.Style

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
		Faint:    lipgloss.NewStyle().Foreground(ColorFaint),

		Accent:  lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),
		Success: lipgloss.NewStyle().Foreground(ColorSuccess),
		Warn:    lipgloss.NewStyle().Foreground(ColorWarn),
		Danger:  lipgloss.NewStyle().Foreground(ColorDanger),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1),
		PanelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1),
		Overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Background(ColorSurface).
			Padding(1, 3),

		Header: lipgloss.NewStyle().
			Foreground(ColorFG).
			Bold(true).
			Padding(0, 1),
		Footer: lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1),
		KeyHint: lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),
		KeyDesc: lipgloss.NewStyle().Foreground(ColorMuted),
		Spinner: lipgloss.NewStyle().Foreground(ColorAccent),

		StepDone:     lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true),
		StepActive:   lipgloss.NewStyle().Foreground(ColorOnAccent).Background(ColorAccent).Bold(true).Padding(0, 1),
		StepUpcoming: lipgloss.NewStyle().Foreground(ColorFaint),
		StepSep:      lipgloss.NewStyle().Foreground(ColorFaint),

		ChatUser:      lipgloss.NewStyle().Foreground(ColorFG).BorderLeft(true).Border(lipgloss.ThickBorder(), false, false, false, true).BorderForeground(ColorAccent).PaddingLeft(1),
		ChatUserLabel: lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),
		ChatAssistLbl: lipgloss.NewStyle().Foreground(ColorMuted).Bold(true),
		ToolBadge:     lipgloss.NewStyle().Foreground(ColorAccentDim).Bold(true),

		StatusBar: lipgloss.NewStyle().Foreground(ColorMuted),
		Badge:     lipgloss.NewStyle().Foreground(ColorMuted).Background(ColorSurface2).Padding(0, 1),
		CostBadge: lipgloss.NewStyle().Foreground(ColorAccent),
		ScrollHl:  lipgloss.NewStyle().Foreground(ColorFaint),

		ComposerFocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorAccent).Padding(0, 1),
		ComposerBlurred: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder).Padding(0, 1),

		ListItem:     lipgloss.NewStyle().Foreground(ColorFG).Padding(0, 1),
		ListSelected: lipgloss.NewStyle().Foreground(ColorOnAccent).Background(ColorAccent).Bold(true).Padding(0, 1),

		ActionKey: lipgloss.NewStyle().Foreground(ColorAccent).Bold(true),

		Wordmark: "XOCODE",
	}
}
