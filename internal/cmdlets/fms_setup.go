//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	fmsSetupCmd = &cobra.Command{
		Use: "setup",
		Short: "provides all one-time setup commands",
		Long: fmsSetupCmdLongDocs,
	}

	fmsSetupCmdLongDocs = `The FMS setup procedure involves a number of setup tasks that are performed once per competition.  Those commands are grouped under this menu.`
)

func init() {
	fmsCmd.AddCommand(fmsSetupCmd)
}
