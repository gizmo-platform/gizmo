//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms/system"
)

var (
	fmsSystemSetupCmd = &cobra.Command{
		Use:    "system-setup",
		Short:  "Setup or synchronize configuration data for the FMS workstation",
		Long:   fmsSystemSetupCmdLongDocs,
		Run:    fmsSystemSetupCmdRun,
		Hidden: true,
	}

	fmsSystemSetupCmdLongDocs = `The Field Management System (FMS) sits on top of a conventional Void Linux installation.  This command configures users and passwords, and provisions the system image to be the field management system.`
)

func init() {
	fmsCmd.AddCommand(fmsSystemSetupCmd)
}

func fmsSystemSetupCmdRun(c *cobra.Command, args []string) {
	initLogger("system-setup")

	setuptool := system.NewSetupTool(appLogger)
	if err := setuptool.Configure(); err != nil {
		appLogger.Error("Fatal Error during install", "error", err)
	}
}
