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
	fmsWizardCmd = &cobra.Command{
		Use:   "wizard",
		Short: "The wizard will walk you through configuration of your field system.",
		Long:  fmsWizardCmdLongDocs,
		Run:   fmsWizardCmdRun,
	}

	fmsWizardCmdLongDocs = `The wizard is a utility that configures the FMS based on your competition team roster and number of fields.  It is a guided process that will prompt you for required information and provide you an opportunity to review the configuration prior to writing it out to disk.`
)

func init() {
	fmsCmd.AddCommand(fmsWizardCmd)
}

func fmsWizardCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		cfg, err := fms.WizardSurvey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running the wizard! (%s)\n", err)
			return 1
		}

		f, err := os.Create("fms.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening fms.json: %s\n", err)
			return 1
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %s\n", err)
			return 2
		}

		return 0
	}())
}
