//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	fmsWizardCmd = &cobra.Command{
		Use:   "wizard",
		Short: "The wizard will walk you through configuration of your field system.",
		Long:  fmsWizardCmdLongDocs,
		Run:   fmsWizardCmdRun,
	}

	fmsWizardCmdLongDocs = `The wizard is a utility that configures the FMS based on your competition team roster and number of fields.  It is a guided process that will prompt you for required information and provide you an opportunity to review the configuration prior to writing it out to disk.`
)

func init() {
	fmsSetupCmd.AddCommand(fmsWizardCmd)
}

func fmsWizardCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		cfg, err := config.NewFMSConfig(nil)
		if err := cfg.WizardSurvey(err == nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error running the wizard! (%s)\n", err)
			return 1
		}

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error waving fms.json: %s\n", err)
			return 1
		}

		return 0
	}())
}
