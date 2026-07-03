package tui

import (
	"time"

	"github.com/xogent/xocode/internal/doctor"
	"github.com/xogent/xocode/internal/plan"
	"github.com/xogent/xocode/internal/stream"
)

// preflightDoneMsg carries the result of the prerequisite checks (initial run
// or a re-check after remediation).
type preflightDoneMsg struct{ results []doctor.Result }

// channelReadyMsg is emitted once a subprocess has been launched and its event
// channel is ready to be pumped. turn tags the stream so stale events from a
// cancelled turn can be dropped.
type channelReadyMsg struct {
	ch    <-chan stream.StreamEvent
	phase State
	turn  int
}

// streamEventMsg carries a single normalized event from the active subprocess.
type streamEventMsg struct {
	ev   stream.StreamEvent
	turn int
}

// streamEOFMsg fires when the active subprocess's event channel closes.
type streamEOFMsg struct {
	phase State
	turn  int
}

// editorFinishedMsg fires after the external $EDITOR process exits.
type editorFinishedMsg struct{ err error }

// plansLoadedMsg carries the saved plans for the history browser.
type plansLoadedMsg struct {
	plans []plan.Plan
	err   error
}

// tickMsg drives the elapsed-time display while a turn is streaming.
type tickMsg time.Time

// errMsg carries a terminal error for the Error state.
type errMsg struct{ err error }
