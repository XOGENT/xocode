package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xogent/xocode/internal/doctor"
	"github.com/xogent/xocode/internal/runner"
	"github.com/xogent/xocode/internal/stream"
)

// runChecks runs the prerequisite checks off the UI thread.
func runChecks(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		return preflightDoneMsg{results: doctor.RunAll(ctx)}
	}
}

// remediate builds a sequence that installs missing tools and logs in to any
// that aren't authenticated, each attached to the terminal, then re-checks.
func remediate(ctx context.Context, results []doctor.Result) tea.Cmd {
	var steps []tea.Cmd
	for _, r := range results {
		if !r.Installed {
			steps = append(steps, doctor.InstallCmd(r.InstallCmd))
		}
	}
	for _, r := range results {
		// Re-resolve login need after installs by checking the current result;
		// the re-check at the end confirms the true state regardless.
		if r.Installed && !r.LoggedIn {
			steps = append(steps, doctor.LoginCmd(r.Bin, r.LoginArgs...))
		}
	}
	steps = append(steps, runChecks(ctx))
	return tea.Sequence(steps...)
}

// startPlanning launches claude in read-only plan mode and hands back the event
// channel. The heavy work (spawning the process) happens off the UI thread.
func startPlanning(ctx context.Context, workdir, model, effort, task string) tea.Cmd {
	return func() tea.Msg {
		cmd := runner.BuildClaudeCmd(ctx, runner.ClaudeSpec{
			Task:    task,
			Model:   model,
			Effort:  effort,
			Workdir: workdir,
		})
		ch, err := stream.NewRunner(cmd, stream.ClaudeAdapter{}).Start(ctx)
		if err != nil {
			return errMsg{err}
		}
		return channelReadyMsg{ch: ch, phase: StatePlanning}
	}
}

// startBuilding launches cursor-agent (Composer 2.5) against the approved plan
// in an isolated worktree, and hands back the event channel.
func startBuilding(ctx context.Context, workdir, model, worktree, prompt string) tea.Cmd {
	return func() tea.Msg {
		cmd := runner.BuildCursorCmd(ctx, runner.CursorSpec{
			Prompt:   prompt,
			Model:    model,
			Worktree: worktree,
			Workdir:  workdir,
		})
		ch, err := stream.NewRunner(cmd, stream.CursorAdapter{}).Start(ctx)
		if err != nil {
			return errMsg{err}
		}
		return channelReadyMsg{ch: ch, phase: StateBuilding}
	}
}

// buildPrompt wraps the plan text with an instruction for the build agent.
func buildPrompt(planText string) string {
	return "Implement the following plan in this repository. Make all necessary " +
		"code changes, then briefly summarize what you did.\n\n" + planText
}

// waitForEvent reads exactly one event off the channel and returns it as a
// message. The Update loop re-issues this after handling each event, so the
// event loop never blocks and the buffered channel provides natural
// back-pressure.
func waitForEvent(ch <-chan stream.StreamEvent, phase State) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamEOFMsg{phase: phase}
		}
		return streamEventMsg{ev: ev}
	}
}
