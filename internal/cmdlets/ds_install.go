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
	if err := ds.New().Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during install: %s\n", err)
		return
	}
}
