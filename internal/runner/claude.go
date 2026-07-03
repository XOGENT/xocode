package runner

import (
	"context"
	"os/exec"
)

// planProtocol is appended to Claude's system prompt on every planning turn. It
// makes plan detection deterministic and environment-independent: Claude wraps a
// finished, approvable plan in the sentinel markers and replies normally to
// anything else. This is what stops a greeting like "hi" from being mistaken for
// a plan — ExitPlanMode is not exposed to headless `claude -p` in current builds,
// so we cannot rely on it.
const planProtocol = `PLANNING PROTOCOL (follow exactly):
- You are collaborating with the user to produce an implementation plan. Investigate the codebase as needed (read-only).
- If the user's message is a greeting, small talk, a question, or needs clarification, reply normally in plain prose. Do NOT produce a plan.
- Only when you have a COMPLETE, actionable implementation plan the user can approve and build, output the plan as GitHub-flavored Markdown wrapped between a line containing only <<<XOCODE_PLAN>>> and a line containing only <<<XOCODE_PLAN_END>>>. Emit these markers nowhere else, and do not write the plan to any file.
- A good plan states the goal, the files to change, the approach, and how to verify it.`

// ClaudeSpec parameterizes a plan-phase invocation of the Claude Code CLI.
type ClaudeSpec struct {
	Task      string // the user's message for this turn
	Model     string // e.g. "opus" (alias for Opus 4.8) or "claude-opus-4-8"
	Effort    string // e.g. "high"
	Workdir   string // directory to run in (read-only plan mode)
	SessionID string // stable session id for the planning conversation
	Resume    bool   // true for follow-up turns (resume the existing session)
}

// BuildClaudeCmd constructs a planning-conversation turn. The first turn opens a
// named session; follow-ups resume it so the conversation keeps its context:
//
//	// first turn
//	claude -p "<msg>" --model opus --effort high --permission-mode plan \
//	       --append-system-prompt "<protocol>" --session-id <uuid> \
//	       --output-format stream-json --verbose
//	// follow-up turns
//	claude -p "<msg>" --permission-mode plan --append-system-prompt "<protocol>" \
//	       --resume <uuid> --output-format stream-json --verbose
//
// stream-json requires --verbose; --permission-mode plan keeps it read-only.
func BuildClaudeCmd(ctx context.Context, s ClaudeSpec) *exec.Cmd {
	args := []string{"-p", s.Task}
	if s.Resume && s.SessionID != "" {
		// Model/effort are fixed by the session at creation time.
		args = append(args, "--resume", s.SessionID)
	} else {
		args = append(args, "--model", s.Model, "--effort", s.Effort)
		if s.SessionID != "" {
			args = append(args, "--session-id", s.SessionID)
		}
	}
	args = append(args,
		"--permission-mode", "plan",
		"--append-system-prompt", planProtocol,
		"--output-format", "stream-json",
		"--verbose",
	)
	return prepare(exec.CommandContext(ctx, "claude", args...), s.Workdir)
}
