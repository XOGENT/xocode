// Package cli wires up the cobra command tree for xocode.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/xogent/xocode/internal/tui"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "xocode",
		Short: "Plan with Opus 4.8, build with Composer 2.5 — from one terminal.",
		Long: "xocode orchestrates the Claude Code and Cursor CLIs into a\n" +
			"plan → review → build workflow. Running `xocode` with no\n" +
			"subcommand launches the interactive TUI.",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Bare `xocode` launches the TUI.
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newUpgradeCmd())
	return root
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "xocode:", err)
		return 1
	}
	return 0
}
