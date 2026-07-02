package cli

import (
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update xocode to the latest release",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd, checkOnly)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "report whether an update is available without installing")
	return cmd
}
