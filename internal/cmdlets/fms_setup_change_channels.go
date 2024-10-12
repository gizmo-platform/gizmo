//go:build linux

package cmdlets

import (
	"encoding/json"
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
		fmsConf, err := fms.LoadConfig("fms.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not load fms.json, have you run the wizard yet? (%s)\n", err)
			return 1
		}

		fmsConf, err = fms.WizardChangeChannels(fmsConf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not change roster information: %s\n", err)
			return 1
		}

		f, err := os.Create("fms.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening fms.json: %s\n", err)
			return 1
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(fmsConf); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %s\n", err)
			return 2
		}
		return 0

	}())
}
