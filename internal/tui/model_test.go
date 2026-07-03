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

// fakePlan streams a canned planning turn that ends with a real plan
// (KindPlan), exercising the full pump without spawning claude.
func fakePlan(planBody string) func(ctx context.Context, sessionID, message string, resume bool, turn int) tea.Cmd {
	return func(ctx context.Context, sessionID, message string, resume bool, turn int) tea.Cmd {
		return func() tea.Msg {
			ch := make(chan stream.StreamEvent, 5)
			ch <- stream.StreamEvent{Kind: stream.KindSystemInit, SessionID: sessionID}
			ch <- stream.StreamEvent{Kind: stream.KindAssistantText, Text: "Exploring the codebase"}
			ch <- stream.StreamEvent{Kind: stream.KindToolUse, ToolName: "Read", ToolInfo: "main.go"}
			ch <- stream.StreamEvent{Kind: stream.KindPlan, Text: planBody}
			ch <- stream.StreamEvent{Kind: stream.KindResult, Text: "done"}
			close(ch)
			return channelReadyMsg{ch: ch, phase: StatePlanning, turn: turn}
		}
	}
}

func newTestModel(t *testing.T, planBody string) Model {
	t.Helper()
	m := NewModel()
	m.state = StateInput // bypass the preflight gate in tests
	m.checking = false
	m.startPlan = fakePlan(planBody)
	m.store = plan.NewStore(t.TempDir()) // keep plan files out of the repo
	return m
}

func TestInputToReviewFlow(t *testing.T) {
	const planText = "PLAN: build the thing in three steps."
	m := newTestModel(t, planText)
	storeDir := m.store.Dir()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Add a hello command")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return contains(b, "three steps")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	entries, _ := os.ReadDir(storeDir)
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".md") {
		t.Fatalf("expected one .md plan file in %s, got %v", storeDir, entries)
	}
}

// TestConversationalTurnStaysInPlanning is the regression guard for the headline
// bug: a greeting must NOT be saved as a plan or advance to review.
func TestConversationalTurnStaysInPlanning(t *testing.T) {
	m := newTestModel(t, "unused")
	m.width, m.height = 120, 40
	m.layout()
	storeDir := m.store.Dir()

	updated, _ := m.beginPlanning("hi")
	m = updated.(Model)

	// Simulate a conversational stream: assistant reply + result, no plan.
	(&m).applyEvent(stream.StreamEvent{Kind: stream.KindAssistantText, Text: "Hi! What can I build?"})
	(&m).applyEvent(stream.StreamEvent{Kind: stream.KindResult, Text: "Hi! What can I build?"})
	res, _ := m.handleEOF(streamEOFMsg{phase: StatePlanning, turn: m.turn})
	m = res.(Model)

	if m.state != StatePlanning {
		t.Fatalf("expected to stay in StatePlanning after a chat reply, got %v", m.state)
	}
	if m.planCaptured {
		t.Fatal("a conversational reply must not be captured as a plan")
	}
	if entries, _ := os.ReadDir(storeDir); len(entries) != 0 {
		t.Fatalf("no plan file should be written for a chat reply, got %v", entries)
	}
}

// TestSentinelResultIsCapturedAsPlan verifies the result-fallback path: a result
// carrying the plan markers is captured and advances to review.
func TestSentinelResultIsCapturedAsPlan(t *testing.T) {
	m := newTestModel(t, "unused")
	m.width, m.height = 120, 40
	m.layout()

	updated, _ := m.beginPlanning("add a flag")
	m = updated.(Model)

	body := "# Plan\n\nDo the thing."
	(&m).applyEvent(stream.StreamEvent{
		Kind: stream.KindResult,
		Text: "Here you go:\n" + stream.PlanMarkerBegin + "\n" + body + "\n" + stream.PlanMarkerEnd,
	})
	res, _ := m.handleEOF(streamEOFMsg{phase: StatePlanning, turn: m.turn})
	m = res.(Model)

	if m.state != StateReview {
		t.Fatalf("expected StateReview after a sentinel plan, got %v", m.state)
	}
	if !strings.Contains(m.planText, "Do the thing") {
		t.Fatalf("plan text not captured: %q", m.planText)
	}
}

func TestEmptySubmitStaysInInput(t *testing.T) {
	m := newTestModel(t, "unused")
	updated, _ := m.updateConversation(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.(Model).state != StateInput {
		t.Fatalf("empty submit should stay in StateInput, got %v", updated.(Model).state)
	}
}

func fakeBuild(final string) func(ctx context.Context, worktree, prompt string, turn int) tea.Cmd {
	return func(ctx context.Context, worktree, prompt string, turn int) tea.Cmd {
		return func() tea.Msg {
			ch := make(chan stream.StreamEvent, 3)
			ch <- stream.StreamEvent{Kind: stream.KindAssistantText, Text: "Editing files"}
			ch <- stream.StreamEvent{Kind: stream.KindResult, Text: final}
			close(ch)
			return channelReadyMsg{ch: ch, phase: StateBuilding, turn: turn}
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
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return contains(b, "PLAN body") },
		teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // approve → build (repo exists, no confirm)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return contains(b, "Build complete")
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
