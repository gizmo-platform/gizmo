//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms/system"
)

var (
	fmsSystemInstallCmd = &cobra.Command{
		Use:    "system-install",
		Short:  "Install packages and retrieve components that are needed from the internet",
		Run:    fmsSystemInstallCmdRun,
		Hidden: true,
	}
)

func init() {
	fmsCmd.AddCommand(fmsSystemInstallCmd)
}

func fmsSystemInstallCmdRun(c *cobra.Command, args []string) {
	initLogger("system-install")

	setuptool := system.NewSetupTool(appLogger)
	if err := setuptool.Install(); err != nil {
		appLogger.Error("Fatal Error during install", "error", err)
	}

	if err := setuptool.SetupGizmoFMSSvc(); err != nil {
		appLogger.Error("Fatal Error during system service setup", "error", err)
	}

	if err := setuptool.SetupBoot(); err != nil {
		appLogger.Error("Fatal Error during boot setup", "error", err)
	}
}
