//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/ds"
)

var (
	dsInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "install attempts to install all system packages",
		Run:   dsInstallCmdRun,
	}
)

func init() {
	dsCmd.AddCommand(dsInstallCmd)
}

func dsInstallCmdRun(c *cobra.Command, args []string) {
	drv := ds.New()

	if err := drv.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during install: %s\n", err)
		return
	}

	if err := drv.SetupBoot(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during boot setup: %s\n", err)
		return
	}
}
