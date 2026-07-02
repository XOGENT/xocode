package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/xogent/xocode/internal/plan"
	"github.com/xogent/xocode/internal/stream"
)

// fakePlan returns a startPlan function that streams a canned sequence of
// events (assistant text, a tool use, then a final result) so the full pump
// (channelReadyMsg → waitForEvent → applyEvent → EOF → Review) is exercised
// without spawning the real claude CLI.
func fakePlan(final string) func(ctx context.Context, task string) tea.Cmd {
	return func(ctx context.Context, task string) tea.Cmd {
		return func() tea.Msg {
			ch := make(chan stream.StreamEvent, 4)
			ch <- stream.StreamEvent{Kind: stream.KindAssistantText, Text: "Exploring the codebase"}
			ch <- stream.StreamEvent{Kind: stream.KindToolUse, ToolName: "Read", ToolInfo: "main.go"}
			ch <- stream.StreamEvent{Kind: stream.KindResult, Text: final}
			close(ch)
			return channelReadyMsg{ch: ch, phase: StatePlanning}
		}
	}
}

func newTestModel(t *testing.T, final string) Model {
	t.Helper()
	m := NewModel()
	m.state = StateInput // bypass the preflight gate in tests
	m.checking = false
	m.startPlan = fakePlan(final)
	m.store = plan.NewStore(t.TempDir()) // keep plan files out of the repo
	return m
}

func TestInputToReviewFlow(t *testing.T) {
	const planText = "PLAN: build the thing in three steps."
	m := newTestModel(t, planText)
	storeDir := m.store.Dir()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Type a task and submit with ctrl+d.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Add a hello command")})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	// The plan text should surface in the review viewport.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return contains(b, "three steps")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// A plan document should have been written.
	entries, _ := os.ReadDir(storeDir)
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".md") {
		t.Fatalf("expected one .md plan file in %s, got %v", storeDir, entries)
	}
}

func TestEmptyTaskDoesNotSubmit(t *testing.T) {
	m := newTestModel(t, "unused")
	// ctrl+d on an empty textarea must stay in the input state.
	updated, _ := m.updateInput(tea.KeyMsg{Type: tea.KeyCtrlD})
	if updated.(Model).state != StateInput {
		t.Fatalf("empty submit should stay in StateInput, got %v", updated.(Model).state)
	}
}

func fakeBuild(final string) func(ctx context.Context, worktree, prompt string) tea.Cmd {
	return func(ctx context.Context, worktree, prompt string) tea.Cmd {
		return func() tea.Msg {
			ch := make(chan stream.StreamEvent, 3)
			ch <- stream.StreamEvent{Kind: stream.KindAssistantText, Text: "Editing files"}
			ch <- stream.StreamEvent{Kind: stream.KindResult, Text: final}
			close(ch)
			return channelReadyMsg{ch: ch, phase: StateBuilding}
		}
	}
}

func TestReviewToBuildSummary(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	m := newTestModel(t, "PLAN body")
	m.projectDir = repo
	m.startBuild = fakeBuild("Implemented in 2 files.")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("do the thing")})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return contains(b, "PLAN body") },
		teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // approve → build (repo exists, no confirm)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return contains(b, "Build complete") && contains(b, "worktrees")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestNonRepoTriggersConfirmInit(t *testing.T) {
	m := newTestModel(t, "PLAN body")
	m.projectDir = t.TempDir() // not a git repo
	updated, _ := m.beginBuilding()
	if updated.(Model).state != StateConfirmInit {
		t.Fatalf("expected StateConfirmInit for non-repo dir, got %v", updated.(Model).state)
	}
}

func contains(haystack []byte, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(h []byte, n string) int {
	nb := []byte(n)
outer:
	for i := 0; i+len(nb) <= len(h); i++ {
		for j := 0; j < len(nb); j++ {
			if h[i+j] != nb[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
