package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	debugCmd = &cobra.Command{
		Use:    "debug",
		Hidden: true,
	}
)

func init() {
	rootCmd.AddCommand(debugCmd)
}
