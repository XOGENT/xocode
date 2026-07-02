package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/xogent/xocode/internal/doctor"
	"github.com/xogent/xocode/internal/selfupdate"
)

var errPrereqs = errors.New("prerequisites not met")

func runDoctor(cmd *cobra.Command, asJSON bool) error {
	results := doctor.RunAll(cmd.Context())
	out := cmd.OutOrStdout()

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return err
		}
		if !doctor.AllOK(results) {
			return errPrereqs
		}
		return nil
	}

	fmt.Fprintln(out, "xocode doctor")
	fmt.Fprintln(out)
	for _, r := range results {
		mark := "✗"
		status := r.Fix
		switch {
		case r.OK():
			mark = "✓"
			status = "installed, logged in"
			if r.Detail != "" {
				status += " (" + r.Detail + ")"
			}
		case r.Installed && !r.LoggedIn:
			mark = "•"
		}
		fmt.Fprintf(out, "  %s %-13s %s\n", mark, r.Name, status)
	}
	fmt.Fprintln(out)

	if !doctor.AllOK(results) {
		fmt.Fprintln(out, "Run `xocode` to install and log in interactively.")
		return errPrereqs
	}
	fmt.Fprintln(out, "All prerequisites satisfied.")
	return nil
}

func runUpgrade(cmd *cobra.Command, checkOnly bool) error {
	return selfupdate.Run(cmd.Context(), checkOnly, cmd.OutOrStdout())
}
