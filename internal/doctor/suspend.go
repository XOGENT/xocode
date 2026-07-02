package doctor

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// StepDoneMsg is emitted after a suspended remediation process exits.
type StepDoneMsg struct{ Err error }

// InstallCmd suspends the TUI and runs an installer one-liner attached to the
// terminal so its progress is visible.
func InstallCmd(oneLiner string) tea.Cmd {
	c := exec.Command("sh", "-c", oneLiner) //nolint:gosec // fixed official installer strings
	return tea.ExecProcess(c, func(err error) tea.Msg { return StepDoneMsg{Err: err} })
}

// LoginCmd suspends the TUI and runs a CLI login attached to the terminal. When
// no display is available (headless/SSH), NO_OPEN_BROWSER is set so the CLI
// prints a URL/code instead of trying to open a browser.
func LoginCmd(bin string, args ...string) tea.Cmd {
	c := exec.Command(bin, args...)
	c.Env = os.Environ()
	if headless() {
		c.Env = append(c.Env, "NO_OPEN_BROWSER=1")
	}
	return tea.ExecProcess(c, func(err error) tea.Msg { return StepDoneMsg{Err: err} })
}

func headless() bool {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return false
	}
	// On macOS there is no DISPLAY, but a browser can still open; only treat
	// remote sessions as headless.
	return os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != ""
}
