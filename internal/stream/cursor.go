package stream

// CursorAdapter parses `cursor-agent --output-format stream-json` output.
// cursor additionally emits a `user` echo line, which parseCommon ignores via
// its default case.
type CursorAdapter struct{}

func (CursorAdapter) Parse(line []byte) (StreamEvent, bool) {
	return parseCommon(line)
}
