//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
)

var (
	fmsSetupChangeChannelCmd = &cobra.Command{
		Use:   "change-channels",
		Short: "Change the channel each field is pinned to",
		Run:   fmsSetupChangeChannelCmdRun,
	}
)

func init() {
	fmsSetupCmd.AddCommand(fmsSetupChangeChannelCmd)
}

func fmsSetupChangeChannelCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		fmsConf, err := fms.NewConfig(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not load fms.json, have you run the wizard yet? (%s)\n", err)
			return 1
		}

		if fmsConf.WizardChangeChannels(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not change roster information: %s\n", err)
			return 1
		}

		if err := fmsConf.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save config: %s\n", err)
			return 1
		}
		return 0

	}())
}
