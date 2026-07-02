package tui

import (
	"github.com/xogent/xocode/internal/doctor"
	"github.com/xogent/xocode/internal/stream"
)

// preflightDoneMsg carries the result of the prerequisite checks (initial run
// or a re-check after remediation).
type preflightDoneMsg struct{ results []doctor.Result }

// channelReadyMsg is emitted once a subprocess has been launched and its event
// channel is ready to be pumped.
type channelReadyMsg struct {
	ch    <-chan stream.StreamEvent
	phase State
}

// streamEventMsg carries a single normalized event from the active subprocess.
type streamEventMsg struct{ ev stream.StreamEvent }

// streamEOFMsg fires when the active subprocess's event channel closes.
type streamEOFMsg struct {
	phase State
	err   error // non-nil if the process errored
}

// planReadyMsg signals the plan document has been persisted.
type planReadyMsg struct {
	path string
	text string
}

// editorFinishedMsg fires after the external $EDITOR process exits.
type editorFinishedMsg struct{ err error }

// errMsg carries a terminal error for the Error state.
type errMsg struct{ err error }
