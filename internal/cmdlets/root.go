// Package cmdlets contains the main entrypoints of the various
// functions that the gizmo tool can perform.
package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gizmo",
		Short: "Entrypoint for all BEST Robot commands",
		Long:  rootCmdLongDocs,
	}
	rootCmdLongDocs = `The Gizmo Platform provides servers for field control, configuration for your joysticks, and tools to program the system processor on your robot control board.`
)

// Entrypoint is the entrypoint into all cmdlets, it will dispatch to
// the right one.
func Entrypoint() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}
