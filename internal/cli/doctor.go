package cli

import (
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check that Claude Code and Cursor CLIs are installed and logged in",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implemented in the doctor task; wired here so the command
			// surface is stable from the start.
			return runDoctor(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON and exit non-zero on failure")
	return cmd
}
