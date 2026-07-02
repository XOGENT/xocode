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
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Message struct {
		Content []contentBlock `json:"content"`
	} `json:"message"`
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
		return StreamEvent{Kind: kind, Text: env.Result, Raw: line}, true

	case "assistant":
		var text strings.Builder
		var toolName, toolInfo string
		for _, b := range env.Message.Content {
			switch b.Type {
			case "text":
				text.WriteString(b.Text)
			case "tool_use":
				if toolName == "" { // capture the first tool in the message
					toolName = b.Name
					toolInfo = extractToolInfo(b.Input)
				}
			}
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
		return StreamEvent{Kind: KindSystemInit, Raw: line}, true

	default:
		// user, rate_limit_event, hook_*, and anything else: ignore.
		return StreamEvent{Kind: KindUnknown, Raw: line}, false
	}
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
