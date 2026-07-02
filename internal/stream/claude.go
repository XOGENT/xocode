package stream

// ClaudeAdapter parses `claude --output-format stream-json --verbose` output.
// claude additionally emits hook_* and rate_limit_event lines, which
// parseCommon ignores via its default case.
type ClaudeAdapter struct{}

func (ClaudeAdapter) Parse(line []byte) (StreamEvent, bool) {
	return parseCommon(line)
}
