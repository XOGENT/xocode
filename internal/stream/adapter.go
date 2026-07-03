package stream

import (
	"encoding/json"
	"strings"
)

// Adapter converts one line of a CLI's stream-json output into a StreamEvent.
// ok=false means "ignore this line" (unknown or uninteresting) — a malformed or
// unrecognized line is never fatal.
type Adapter interface {
	Parse(line []byte) (StreamEvent, bool)
}

// envelope is a permissive view over the events emitted by both `claude` and
// `cursor-agent`. Their schemas share the same core shape (verified against
// captured fixtures): a `type` discriminator, a `result` string on the final
// event, and assistant messages carrying `message.content[]` blocks.
type envelope struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	IsError      bool    `json:"is_error"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	DurationMS   int     `json:"duration_ms"`
	Usage        usage   `json:"usage"`
	Message      struct {
		Content []contentBlock `json:"content"`
	} `json:"message"`
}

// usage tolerates both casings: claude emits snake_case
// (input_tokens/output_tokens), cursor emits camelCase (inputTokens/outputTokens).
type usage struct {
	InputTokens   int `json:"input_tokens"`
	OutputTokens  int `json:"output_tokens"`
	InputTokensC  int `json:"inputTokens"`
	OutputTokensC int `json:"outputTokens"`
}

func (u usage) in() int {
	if u.InputTokens != 0 {
		return u.InputTokens
	}
	return u.InputTokensC
}

func (u usage) out() int {
	if u.OutputTokens != 0 {
		return u.OutputTokens
	}
	return u.OutputTokensC
}

type contentBlock struct {
	Type  string          `json:"type"`  // "text" | "tool_use" | …
	Text  string          `json:"text"`  // for text blocks
	Name  string          `json:"name"`  // for tool_use blocks
	Input json.RawMessage `json:"input"` // for tool_use blocks
}

// parseCommon implements the shared parsing both adapters use. It is defensive:
// any unmarshalling failure or unrecognized type degrades to (zero, false)
// rather than panicking.
func parseCommon(line []byte) (StreamEvent, bool) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return StreamEvent{}, false
	}

	switch env.Type {
	case "result":
		kind := KindResult
		if env.IsError || env.Subtype == "error" || strings.HasPrefix(env.Subtype, "error") {
			kind = KindError
		}
		return StreamEvent{
			Kind: kind,
			Text: env.Result,
			Usage: Usage{
				InputTokens:  env.Usage.in(),
				OutputTokens: env.Usage.out(),
				CostUSD:      env.TotalCostUSD,
				DurationMS:   env.DurationMS,
			},
			Raw: line,
		}, true

	case "assistant":
		var text strings.Builder
		var toolName, toolInfo string
		for _, b := range env.Message.Content {
			switch b.Type {
			case "text":
				text.WriteString(b.Text)
			case "tool_use":
				// An ExitPlanMode call is the authoritative "this is a plan"
				// signal in Claude's plan mode; its input.plan holds the plan.
				if strings.EqualFold(b.Name, "ExitPlanMode") {
					if p := extractPlan(b.Input); strings.TrimSpace(p) != "" {
						return StreamEvent{Kind: KindPlan, Text: p, SessionID: env.SessionID, Raw: line}, true
					}
				}
				if toolName == "" { // capture the first tool in the message
					toolName = b.Name
					toolInfo = extractToolInfo(b.Input)
				}
			}
		}
		// A text block carrying the sentinel is the finished plan — surface it
		// as KindPlan so it never lands in the chat transcript as raw text.
		if plan, ok := ExtractPlan(text.String()); ok {
			return StreamEvent{Kind: KindPlan, Text: plan, SessionID: env.SessionID, Raw: line}, true
		}
		if toolName != "" {
			return StreamEvent{
				Kind:     KindToolUse,
				Text:     strings.TrimSpace(text.String()),
				ToolName: toolName,
				ToolInfo: toolInfo,
				Raw:      line,
			}, true
		}
		if t := strings.TrimSpace(text.String()); t != "" {
			return StreamEvent{Kind: KindAssistantText, Text: t, Raw: line}, true
		}
		return StreamEvent{Kind: KindUnknown, Raw: line}, false

	case "system":
		// Only the init event carries the session id; hook_* / progress system
		// lines are noise and must not each masquerade as a session init.
		if env.Subtype != "" && env.Subtype != "init" {
			return StreamEvent{Kind: KindUnknown, Raw: line}, false
		}
		return StreamEvent{Kind: KindSystemInit, SessionID: env.SessionID, Raw: line}, true

	default:
		// user, rate_limit_event, hook_*, and anything else: ignore.
		return StreamEvent{Kind: KindUnknown, Raw: line}, false
	}
}

// extractPlan pulls the plan markdown out of an ExitPlanMode tool_use input
// ({"plan": "..."}). Best-effort; empty on any mismatch so a malformed call
// degrades to a normal tool_use rather than crashing.
func extractPlan(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m struct {
		Plan string `json:"plan"`
	}
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	return m.Plan
}

// extractToolInfo pulls a human-friendly descriptor out of a tool_use input
// object, checking the common keys used by both CLIs. Best-effort; empty is fine.
func extractToolInfo(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	for _, key := range []string{"file_path", "path", "command", "pattern", "url", "query"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
