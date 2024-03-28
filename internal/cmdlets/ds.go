//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	dsCmd = &cobra.Command{
		Use:   "ds",
		Short: "ds cmdlets operate or configure a driver's station",
		Long:  dsCmdLongDocs,
	}

	dsCmdLongDocs = `ds cmdlets provide configuration and operations utilities for managing a driver's station`
)

func init() {
	rootCmd.AddCommand(dsCmd)
}
