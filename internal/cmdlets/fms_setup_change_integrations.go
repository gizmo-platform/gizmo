//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	fmsSetupChangeIntegrationsCmd = &cobra.Command{
		Use:   "change-integrations",
		Short: "Change the enabled set of integrations",
		Run:   fmsSetupChangeIntegrationsCmdRun,
	}
)

func init() {
	fmsSetupCmd.AddCommand(fmsSetupChangeIntegrationsCmd)
}

func fmsSetupChangeIntegrationsCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		fmsConf, err := config.NewFMSConfig(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not load fms.json, have you run the wizard yet? (%s)\n", err)
			return 1
		}

		if err := fmsConf.WizardChangeIntegrations(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not change radio mode: %s\n", err)
			return 1
		}

		if err := fmsConf.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save config: %s\n", err)
			return 1
		}
		return 0
	}())
}
