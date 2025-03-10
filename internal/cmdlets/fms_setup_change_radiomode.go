//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

var (
	fmsSetupChangeRadioModeCmd = &cobra.Command{
		Use:   "change-radio",
		Short: "Change the radio mode in use across the system",
		Run:   fmsSetupChangeRadioModeCmdRun,
	}
)

func init() {
	fmsSetupCmd.AddCommand(fmsSetupChangeRadioModeCmd)
	fmsSetupChangeRadioModeCmd.Flags().Bool("skip-apply", false, "Skip applying changes")
}

func fmsSetupChangeRadioModeCmdRun(c *cobra.Command, args []string) {
	skipApply, _ := c.Flags().GetBool("skip-apply")

	os.Exit(func() int {
		fmsConf, err := fms.NewConfig(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not load fms.json, have you run the wizard yet? (%s)\n", err)
			return 1
		}

		if err := fmsConf.WizardChangeRadioMode(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not change radio mode: %s\n", err)
			return 1
		}

		f, err := os.Create("fms.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening fms.json: %s\n", err)
			return 1
		}
		defer f.Close()

		if err := fmsConf.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %s\n", err)
			return 2
		}

		initLogger("change-radio")
		controller := config.New(config.WithFMS(fmsConf), config.WithLogger(appLogger))
		ctx := make(map[string]interface{})
		ctx["RouterBootstrap"] = false
		ctx["FieldBootstrap"] = false
		if err := controller.SyncState(ctx); err != nil {
			appLogger.Error("Fatal error synchronizing state", "error", err)
			return 2
		}

		if skipApply {
			return 0
		}

		if err := controller.Converge(false, ""); err != nil {
			appLogger.Error("Fatal error converging state", "error", err)
			return 2
		}

		if err := controller.ReprovisionCAP(); err != nil {
			fmt.Fprintf(os.Stderr, "Error cycling radios: %s\n", err)
			return 2
		}

		return 0
	}())
}
