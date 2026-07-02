// Package doctor verifies (and helps remediate) the prerequisite CLIs xocode
// orchestrates: Claude Code (`claude`) and Cursor (`cursor-agent`).
package doctor

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// Tool identifies a prerequisite CLI.
type Tool int

const (
	ToolClaude Tool = iota
	ToolCursor
)

// Install one-liners (official installers, verified during design).
const (
	claudeInstall = "curl -fsSL https://claude.ai/install.sh | bash"
	cursorInstall = "curl https://cursor.com/install -fsS | bash"
)

// Result is the outcome of checking one tool.
type Result struct {
	Tool      Tool
	Name      string // human name, e.g. "Claude Code"
	Bin       string // resolved binary name
	Installed bool
	LoggedIn  bool
	Detail    string // e.g. logged-in email
	Fix       string // guidance shown when not OK

	InstallCmd string   // shell one-liner to install
	LoginArgs  []string // args to the resolved bin to log in
}

// OK reports whether the tool is installed and authenticated.
func (r Result) OK() bool { return r.Installed && r.LoggedIn }

// RunAll checks both prerequisites.
func RunAll(ctx context.Context) []Result {
	return []Result{CheckClaude(ctx), CheckCursor(ctx)}
}

// AllOK reports whether every result is OK.
func AllOK(rs []Result) bool {
	for _, r := range rs {
		if !r.OK() {
			return false
		}
	}
	return true
}

// CheckClaude verifies the Claude Code CLI. Auth is read from
// `claude auth status` which prints JSON including {"loggedIn": bool, "email"}.
func CheckClaude(ctx context.Context) Result {
	r := Result{
		Tool: ToolClaude, Name: "Claude Code", Bin: "claude",
		InstallCmd: claudeInstall, LoginArgs: []string{"auth", "login"},
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		r.Fix = "Install the Claude Code CLI"
		return r
	}
	r.Installed = true
	_ = path

	out, err := exec.CommandContext(ctx, "claude", "auth", "status").Output()
	if err == nil {
		var st struct {
			LoggedIn bool   `json:"loggedIn"`
			Email    string `json:"email"`
		}
		if json.Unmarshal(out, &st) == nil && st.LoggedIn {
			r.LoggedIn = true
			r.Detail = st.Email
			return r
		}
	}
	r.Fix = "Log in with: claude auth login"
	return r
}

// CheckCursor verifies the Cursor CLI. Auth is read from `cursor-agent status`
// which prints "Logged in as <email>" when authenticated.
func CheckCursor(ctx context.Context) Result {
	r := Result{
		Tool: ToolCursor, Name: "Cursor", Bin: "cursor-agent",
		InstallCmd: cursorInstall, LoginArgs: []string{"login"},
	}
	bin := "cursor-agent"
	if _, err := exec.LookPath(bin); err != nil {
		if _, err2 := exec.LookPath("agent"); err2 == nil {
			bin = "agent"
		} else {
			r.Fix = "Install the Cursor CLI"
			return r
		}
	}
	r.Bin = bin
	r.Installed = true

	out, err := exec.CommandContext(ctx, bin, "status").CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(out)), "logged in") {
		r.LoggedIn = true
		r.Detail = extractLoginLine(string(out))
		return r
	}
	r.Fix = "Log in with: " + bin + " login"
	return r
}

func extractLoginLine(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(strings.ToLower(line), "logged in") {
			return strings.TrimSpace(strings.TrimLeft(line, "✓ "))
		}
	}
	return ""
}
