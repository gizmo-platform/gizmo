package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	fieldCmd = &cobra.Command{
		Use:   "field",
		Short: "field cmdlets operate or configure a field",
		Long:  fieldCmdLongDocs,
	}

	fieldCmdLongDocs = `field cmdlets provide various configuration utilities for setting up your fields, as well as the core server that connects joysticks to field.`
)

func init() {
	rootCmd.AddCommand(fieldCmd)
}
