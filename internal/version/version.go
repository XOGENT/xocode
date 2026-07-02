// Package version exposes build metadata injected at link time via -ldflags.
package version

import "fmt"

// These are overridden at build time by GoReleaser:
//
//	-X github.com/xogent/xocode/internal/version.Version=v1.2.3
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable version line.
func String() string {
	return fmt.Sprintf("xocode %s (commit %s, built %s)", Version, Commit, Date)
}
