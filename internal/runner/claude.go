package runner

import (
	"context"
	"os/exec"
)

// ClaudeSpec parameterizes a plan-phase invocation of the Claude Code CLI.
type ClaudeSpec struct {
	Task    string // the user's task prompt
	Model   string // e.g. "opus" (alias for Opus 4.8) or "claude-opus-4-8"
	Effort  string // e.g. "high"
	Workdir string // directory to run in (read-only plan mode)
}

// BuildClaudeCmd constructs the read-only planning invocation:
//
//	claude -p "<task>" --model opus --effort high --permission-mode plan \
//	       --output-format stream-json --verbose
//
// stream-json requires --verbose; --permission-mode plan keeps it read-only.
func BuildClaudeCmd(ctx context.Context, s ClaudeSpec) *exec.Cmd {
	args := []string{
		"-p", s.Task,
		"--model", s.Model,
		"--effort", s.Effort,
		"--permission-mode", "plan",
		"--output-format", "stream-json",
		"--verbose",
	}
	return prepare(exec.CommandContext(ctx, "claude", args...), s.Workdir)
}
