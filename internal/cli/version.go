package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/xogent/xocode/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the xocode version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version.String())
			return nil
		},
	}
}
