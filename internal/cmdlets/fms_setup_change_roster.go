//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	fmsSetupChangeRosterCmd = &cobra.Command{
		Use:   "change-roster",
		Short: "Change the team roster",
		Long:  fmsSetupChangeRosterCmdLongDocs,
		Run:   fmsSetupChangeRosterCmdRun,
	}

	fmsSetupChangeRosterCmdLongDocs = ``
)

func init() {
	fmsSetupCmd.AddCommand(fmsSetupChangeRosterCmd)
}

func fmsSetupChangeRosterCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		fmsConf, err := config.NewFMSConfig(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not load fms.json, have you run the wizard yet? (%s)\n", err)
			return 1
		}

		if err := fmsConf.WizardChangeRoster(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not change roster information: %s\n", err)
			return 1
		}

		if err := fmsConf.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save FMS Config: %s\n", err)
			return 1
		}

		return 0
	}())
}
