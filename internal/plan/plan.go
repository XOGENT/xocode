// Package plan handles persistence of the plan documents produced during the
// planning phase.
package plan

import "time"

// Plan is a generated implementation plan plus its metadata.
type Plan struct {
	Task      string
	Slug      string
	Path      string // absolute path once saved
	Text      string // the plan body (authoritative, from claude's result)
	Model     string // e.g. "claude-opus-4-8" / "opus"
	CreatedAt time.Time
}
