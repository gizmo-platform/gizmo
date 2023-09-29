package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	firmwareCmd = &cobra.Command{
		Use:   "firmware",
		Short: "firmware cmdlets manage the GSS firmware image",
		Long:  firmwareCmdLongDocs,
	}

	firmwareCmdLongDocs = `firmware cmdlets handle the process of creating the Gizmo System Software (GSS) firmware images`
)

func init() {
	rootCmd.AddCommand(firmwareCmd)
}
