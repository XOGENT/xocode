package tui

// State enumerates the phases of the plan → review → build workflow.
type State int

const (
	StatePreflight   State = iota // verifying claude/cursor installed + logged in
	StateInput                    // user types the task prompt
	StatePlanning                 // claude streaming (read-only plan mode)
	StateReview                   // show plan; view/edit/approve
	StateBuilding                 // cursor-agent streaming in a worktree
	StateSummary                  // done: worktree path + merge guidance
	StateConfirmInit              // ask before `git init` in a non-repo directory
	StateError                    // terminal error
)

// label is the breadcrumb text shown in the header for each state.
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
	case StateError:
		return "Error"
	default:
		return ""
	}
}
