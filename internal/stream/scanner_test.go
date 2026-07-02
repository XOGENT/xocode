package stream

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// lineAdapter maps every non-empty line to an assistant-text event.
type lineAdapter struct{}

func (lineAdapter) Parse(line []byte) (StreamEvent, bool) {
	return StreamEvent{Kind: KindAssistantText, Text: string(line)}, true
}

func TestScannerStreamsAndClosesOnExit(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "printf 'a\\nb\\nc\\n'")
	ch, err := NewRunner(cmd, lineAdapter{}).Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	var got []string
	for ev := range ch {
		if ev.Kind == KindAssistantText {
			got = append(got, ev.Text)
		}
	}
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("unexpected events: %v", got)
	}
}

func TestScannerCancellationClosesChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Emit one line, then block for a long time.
	cmd := exec.CommandContext(ctx, "sh", "-c", "printf 'first\\n'; sleep 30")
	cmd.Cancel = func() error { return cmd.Process.Kill() }
	cmd.WaitDelay = time.Second
	ch, err := NewRunner(cmd, lineAdapter{}).Start(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if ev := <-ch; ev.Text != "first" {
		t.Fatalf("expected 'first', got %q", ev.Text)
	}
	cancel()

	// The channel must close promptly after cancellation.
	select {
	case _, ok := <-ch:
		// Either a trailing event then close, or immediate close; drain to close.
		if ok {
			for range ch {
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel did not close within 5s of cancellation")
	}
}
