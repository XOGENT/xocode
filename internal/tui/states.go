package tui

// State enumerates the screens of the plan → review → build workflow.
type State int

const (
	StatePreflight   State = iota // verifying claude/cursor installed + logged in
	StateInput                    // landing: compose the first message
	StatePlanning                 // conversation with claude (streaming or idle)
	StateReview                   // show plan; approve / edit / refine / discard
	StateBuilding                 // cursor-agent streaming in a worktree
	StateSummary                  // done: worktree path + merge guidance
	StateConfirmInit              // ask before `git init` in a non-repo directory
	StateHistory                  // browse & reopen saved plans
	StateSettings                 // edit models / effort
	StateError                    // terminal error
)

// label is the short breadcrumb text for a state.
func (s State) label() string {
	switch s {
	case StatePreflight:
		return "Setup"
	case StateInput:
		return "Task"
	case StatePlanning:
		return "Planning"
	case StateReview:
		return "Review"
	case StateBuilding:
		return "Building"
	case StateSummary:
		return "Done"
	case StateConfirmInit:
		return "Setup"
	case StateHistory:
		return "History"
	case StateSettings:
		return "Settings"
	case StateError:
		return "Error"
	default:
		return ""
	}
}

// steps are the phases shown in the progress stepper header, in order.
var steps = []struct {
	label string
	state State
}{
	{"Task", StateInput},
	{"Plan", StatePlanning},
	{"Review", StateReview},
	{"Build", StateBuilding},
	{"Done", StateSummary},
}

// stepIndex maps a state to its position in the stepper (−1 for states that sit
// outside the main flow, like history/settings/error).
func (s State) stepIndex() int {
	switch s {
	case StateInput:
		return 0
	case StatePlanning:
		return 1
	case StateReview:
		return 2
	case StateBuilding:
		return 3
	case StateSummary:
		return 4
	default:
		return -1
	}
}
