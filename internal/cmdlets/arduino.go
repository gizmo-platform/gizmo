package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	arduinoCmd = &cobra.Command{
		Use:   "arduino",
		Short: "Configure Arduino tools for use with Gizmo",
		Long:  arduinoCmdLongDocs,
	}

	arduinoCmdLongDocs = `arduino cmdlets can configure and install components of the Arduino Suite for you.  In order for these functions to perform correctly, you must have the arduino-cli installed already.`
)

func init() {
	rootCmd.AddCommand(arduinoCmd)
}
