// Package stream parses the line-delimited JSON emitted by `claude` and
// `cursor-agent` (both with --output-format stream-json) into a single
// normalized event type, and runs the subprocesses that produce it.
package stream

import "encoding/json"

// EventKind classifies a normalized stream event.
type EventKind int

const (
	KindUnknown       EventKind = iota
	KindSystemInit              // session metadata (model, session id)
	KindAssistantText           // assistant narration text
	KindToolUse                 // a tool invocation (Read, Edit, Bash, …)
	KindResult                  // FINAL result payload; Text holds the plan / summary
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
	case KindResult:
		return "result"
	case KindError:
		return "error"
	default:
		return "unknown"
	}
}

// StreamEvent is the vendor-neutral event the TUI consumes. Both CLI adapters
// produce this type, so the TUI's handling is identical for planning and
// building.
type StreamEvent struct {
	Kind     EventKind
	Text     string          // assistant text, or final result text for KindResult
	ToolName string          // for KindToolUse
	ToolInfo string          // for KindToolUse: file path / command / pattern, best-effort
	Partial  bool            // reserved for streamed deltas
	Raw      json.RawMessage // original line, kept for debugging
}
