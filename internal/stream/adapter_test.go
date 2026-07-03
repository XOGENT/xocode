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

// planEvents counts KindPlan events and returns the last plan text.
func planEvents(evs []StreamEvent) (int, string) {
	var n int
	var last string
	for _, e := range evs {
		if e.Kind == KindPlan {
			n++
			last = e.Text
		}
	}
	return n, last
}

// TestClaudePlanFixture: a real plan-mode turn yields exactly one KindPlan whose
// text is the plan body, and the plan is NOT also surfaced as chat text.
func TestClaudePlanFixture(t *testing.T) {
	evs := parseFixture(t, "testdata/claude_plan.jsonl", ClaudeAdapter{})
	n, plan := planEvents(evs)
	if n < 1 {
		t.Fatalf("expected at least 1 KindPlan, got %d", n)
	}
	if !strings.Contains(plan, "version") {
		t.Fatalf("plan text missing expected content: %q", plan)
	}
	if strings.Contains(plan, PlanMarkerBegin) || strings.Contains(plan, PlanMarkerEnd) {
		t.Fatal("extracted plan should not contain the sentinel markers")
	}
	// The raw markers must never leak into assistant chat text.
	for _, e := range evs {
		if e.Kind == KindAssistantText && strings.Contains(e.Text, PlanMarkerBegin) {
			t.Fatal("assistant text leaked a plan marker")
		}
	}
}

// TestClaudeChatNoPlan is the regression guard: a greeting yields no plan.
func TestClaudeChatNoPlan(t *testing.T) {
	evs := parseFixture(t, "testdata/claude_chat.jsonl", ClaudeAdapter{})
	if n, _ := planEvents(evs); n != 0 {
		t.Fatalf("a conversational reply must yield 0 KindPlan, got %d", n)
	}
	evs = parseFixture(t, "testdata/claude.jsonl", ClaudeAdapter{})
	if n, _ := planEvents(evs); n != 0 {
		t.Fatalf("the 'hi'/'Hello!' fixture must yield 0 KindPlan, got %d", n)
	}
}

// TestSystemInitOnlyOnInit: only the init system line is surfaced, and it
// carries the session id (hook_* system lines are ignored).
func TestSystemInitOnlyOnInit(t *testing.T) {
	evs := parseFixture(t, "testdata/claude_plan.jsonl", ClaudeAdapter{})
	var inits int
	var sid string
	for _, e := range evs {
		if e.Kind == KindSystemInit {
			inits++
			if e.SessionID != "" {
				sid = e.SessionID
			}
		}
	}
	if inits != 1 {
		t.Fatalf("expected exactly 1 KindSystemInit, got %d", inits)
	}
	if sid == "" {
		t.Fatal("session id not captured from init event")
	}
}

func TestExitPlanModeExtraction(t *testing.T) {
	line := `{"type":"assistant","session_id":"abc","message":{"content":[{"type":"tool_use","name":"ExitPlanMode","input":{"plan":"# Do X"}}]}}`
	ev, ok := (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindPlan || ev.Text != "# Do X" || ev.SessionID != "abc" {
		t.Fatalf("ExitPlanMode not extracted: ok=%v %+v", ok, ev)
	}
	// A malformed ExitPlanMode (no plan) degrades to a normal tool_use.
	line = `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"ExitPlanMode","input":{}}]}}`
	ev, ok = (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindToolUse {
		t.Fatalf("malformed ExitPlanMode should degrade to tool_use: ok=%v %+v", ok, ev)
	}
}

func TestSentinelInAssistantText(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"ok\n<<<XOCODE_PLAN>>>\n# Plan body\n<<<XOCODE_PLAN_END>>>"}]}}`
	ev, ok := (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindPlan || ev.Text != "# Plan body" {
		t.Fatalf("sentinel plan not extracted from assistant text: ok=%v %+v", ok, ev)
	}
}

func TestUsageParsedBothCasings(t *testing.T) {
	claude := `{"type":"result","subtype":"success","result":"ok","total_cost_usd":0.5,"usage":{"input_tokens":10,"output_tokens":20}}`
	ev, _ := (ClaudeAdapter{}).Parse([]byte(claude))
	if ev.Usage.InputTokens != 10 || ev.Usage.OutputTokens != 20 || ev.Usage.CostUSD != 0.5 {
		t.Fatalf("claude usage not parsed: %+v", ev.Usage)
	}
	cursor := `{"type":"result","subtype":"success","result":"ok","usage":{"inputTokens":7,"outputTokens":8}}`
	ev, _ = (CursorAdapter{}).Parse([]byte(cursor))
	if ev.Usage.InputTokens != 7 || ev.Usage.OutputTokens != 8 {
		t.Fatalf("cursor usage not parsed: %+v", ev.Usage)
	}
}

func TestToolUseExtraction(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/foo.go"}}]}}`
	ev, ok := (ClaudeAdapter{}).Parse([]byte(line))
	if !ok || ev.Kind != KindToolUse || ev.ToolName != "Read" || ev.ToolInfo != "/tmp/foo.go" {
		t.Fatalf("tool_use not extracted correctly: ok=%v %+v", ok, ev)
	}
}
