package runner

import (
	"context"
	"os/exec"
)

// cursorBin is the Cursor CLI binary name, with a documented fallback.
const (
	cursorBin      = "cursor-agent"
	cursorBinAlt   = "agent"
	cursorLoginBin = cursorBin
)

// CursorSpec parameterizes a build-phase invocation of the Cursor CLI.
type CursorSpec struct {
	Prompt   string // the approved plan text plus build instructions
	Model    string // e.g. "composer-2.5"
	Worktree string // isolated worktree name passed to -w
	Workdir  string // the repo directory the worktree is based on
}

// BuildCursorCmd constructs the build invocation:
//
//	cursor-agent -p "<prompt>" --model composer-2.5 --force --trust \
//	             -w <worktree> --output-format stream-json
//
// --force runs all tools without prompting; --trust bypasses the workspace
// trust dialog (required in headless); -w isolates work in a git worktree.
func BuildCursorCmd(ctx context.Context, s CursorSpec) *exec.Cmd {
	bin := resolveCursorBin()
	args := []string{
		"-p", s.Prompt,
		"--model", s.Model,
		"--force",
		"--trust",
		"--output-format", "stream-json",
	}
	if s.Worktree != "" {
		args = append(args, "-w", s.Worktree)
	}
	return prepare(exec.CommandContext(ctx, bin, args...), s.Workdir)
}

// resolveCursorBin prefers `cursor-agent` but falls back to `agent` (the docs
// use both names).
func resolveCursorBin() string {
	if _, err := exec.LookPath(cursorBin); err == nil {
		return cursorBin
	}
	if _, err := exec.LookPath(cursorBinAlt); err == nil {
		return cursorBinAlt
	}
	return cursorBin
}
