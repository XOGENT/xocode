// Package stream parses the line-delimited JSON emitted by `claude` and
// `cursor-agent` (both with --output-format stream-json) into a single
// normalized event type, and runs the subprocesses that produce it.
package stream

import (
	"encoding/json"
	"strings"
)

// Plan sentinel markers. xocode instructs Claude (via an appended system prompt)
// to wrap a finished, approvable plan between these two lines and to emit them
// nowhere else. This is the authoritative "this is a plan" signal — it works in
// every environment, unlike the ExitPlanMode tool, which is not exposed to
// headless `claude -p` in current builds. A conversational reply (e.g. to "hi")
// simply omits the markers, so it is never mistaken for a plan.
const (
	PlanMarkerBegin = "<<<XOCODE_PLAN>>>"
	PlanMarkerEnd   = "<<<XOCODE_PLAN_END>>>"
)

// ExtractPlan returns the plan markdown wrapped in the sentinel markers, and
// whether an opening marker was present. If the closing marker is missing (e.g.
// a truncated stream), everything after the opening marker is returned so a
// partial capture is still usable.
func ExtractPlan(text string) (string, bool) {
	i := strings.Index(text, PlanMarkerBegin)
	if i < 0 {
		return "", false
	}
	rest := text[i+len(PlanMarkerBegin):]
	if j := strings.Index(rest, PlanMarkerEnd); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest), true
}

// StripPlanMarkers removes the sentinel markers (and any plan they wrap) from
// text destined for the chat transcript, so raw markers never surface to the
// user even if a stray marker escapes into conversational prose.
func StripPlanMarkers(text string) string {
	for {
		i := strings.Index(text, PlanMarkerBegin)
		if i < 0 {
			break
		}
		rest := text[i+len(PlanMarkerBegin):]
		j := strings.Index(rest, PlanMarkerEnd)
		if j < 0 {
			text = text[:i]
			break
		}
		text = text[:i] + rest[j+len(PlanMarkerEnd):]
	}
	return strings.TrimSpace(text)
}

// EventKind classifies a normalized stream event.
type EventKind int

const (
	KindUnknown       EventKind = iota
	KindSystemInit              // session metadata (model, session id)
	KindAssistantText           // assistant narration text
	KindToolUse                 // a tool invocation (Read, Edit, Bash, …)
	KindPlan                    // an authoritative plan produced via ExitPlanMode
	KindResult                  // FINAL turn payload; Text holds the run summary
	KindError                   // an error (nonzero exit, or result with is_error)
)

func (k EventKind) String() string {
	switch k {
	case KindSystemInit:
		return "system"
	case KindAssistantText:
		return "assistant"
	case KindToolUse:
		return "tool"
	case KindPlan:
		return "plan"
	case KindResult:
		return "result"
	case KindError:
		return "error"
	default:
		return "unknown"
	}
}

// Usage captures best-effort token accounting from a run's final result event.
// Fields are zero when the CLI doesn't report them.
type Usage struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	DurationMS   int
}

// Empty reports whether the usage carries no information worth displaying.
func (u Usage) Empty() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 && u.CostUSD == 0
}

// StreamEvent is the vendor-neutral event the TUI consumes. Both CLI adapters
// produce this type, so the TUI's handling is identical for planning and
// building.
type StreamEvent struct {
	Kind      EventKind
	Text      string          // assistant text, result summary, or plan (KindPlan)
	ToolName  string          // for KindToolUse
	ToolInfo  string          // for KindToolUse: file path / command / pattern, best-effort
	SessionID string          // for KindSystemInit: the session id to resume with
	Usage     Usage           // for KindResult: best-effort token/cost accounting
	Partial   bool            // reserved for streamed deltas
	Raw       json.RawMessage // original line, kept for debugging
}
