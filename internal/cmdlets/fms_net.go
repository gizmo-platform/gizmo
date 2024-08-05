//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	fmsNetCmd = &cobra.Command{
		Use:   "net",
		Short: "provides all network related commands",
		Long:  fmsNetCmdLongDocs,
	}

	fmsNetCmdLongDocs = `The Field Management System (FMS) manages a robust software defined network infrastructure to provide field services.  All the commands related to this network start with this command.`
)

func init() {
	fmsCmd.AddCommand(fmsNetCmd)
}
