package stream

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// parseFixture runs an adapter over a captured .jsonl fixture and returns the
// normalized events (skipping ignored lines).
func parseFixture(t *testing.T, path string, a Adapter) []StreamEvent {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var evs []StreamEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, scanBufInitial), scanBufMax)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if ev, ok := a.Parse([]byte(line)); ok {
			evs = append(evs, ev)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return evs
}

// finalResult returns the text of the last KindResult event, asserting exactly
// one is present — this extraction is load-bearing for saving the plan.
func finalResult(t *testing.T, evs []StreamEvent) string {
	t.Helper()
	var got string
	var n int
	for _, e := range evs {
		if e.Kind == KindResult {
			got = e.Text
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 KindResult, got %d", n)
	}
	return got
}

func TestClaudeAdapterFixture(t *testing.T) {
	evs := parseFixture(t, "testdata/claude.jsonl", ClaudeAdapter{})
	if len(evs) == 0 {
		t.Fatal("no events parsed from claude fixture")
	}
	if got := finalResult(t, evs); got == "" {
		t.Fatal("claude final result text is empty")
	}
	// hook_* / rate_limit_event lines must be ignored, never KindError.
	for _, e := range evs {
		if e.Kind == KindError {
			t.Fatalf("unexpected KindError in successful claude fixture: %q", e.Text)
		}
	}
}

func TestCursorAdapterFixture(t *testing.T) {
	evs := parseFixture(t, "testdata/cursor.jsonl", CursorAdapter{})
	if len(evs) == 0 {
		t.Fatal("no events parsed from cursor fixture")
	}
	if got := finalResult(t, evs); got == "" {
		t.Fatal("cursor final result text is empty")
	}
	// cursor's `user` echo line must be ignored.
	for _, e := range evs {
		if e.Kind == KindError {
			t.Fatalf("unexpected KindError in successful cursor fixture: %q", e.Text)
		}
	}
}

func TestMalformedLinesTolerated(t *testing.T) {
	cases := []string{
		``,                        // empty
		`not json at all`,         // garbage
		`{"type":`,                // truncated
		`{"type":"mystery_kind"}`, // unknown type
		`{}`,                      // no type
	}
	for _, in := range cases {
		if ev, ok := (ClaudeAdapter{}).Parse([]byte(in)); ok {
			t.Fatalf("expected %q to be ignored, got %+v", in, ev)
		}
	}
}

func TestResultIsErrorMapping(t *testing.T) {
	line := `{"type":"result","subtype":"error_max_turns","is_error":true,"result":"boom"}`
	ev, ok := (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindError || ev.Text != "boom" {
		t.Fatalf("expected KindError with text 'boom', got ok=%v %+v", ok, ev)
	}
}

func TestToolUseExtraction(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/foo.go"}}]}}`
	ev, ok := (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindToolUse || ev.ToolName != "Read" || ev.ToolInfo != "/tmp/foo.go" {
		t.Fatalf("tool_use not extracted correctly: ok=%v %+v", ok, ev)
	}
}
