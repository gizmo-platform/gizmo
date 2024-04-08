package cmdlets

import (
	"github.com/spf13/cobra"
)

var (
	fmsCmd = &cobra.Command{
		Use:   "fms",
		Short: "provides an entrypoint to the Field Management System hierarchy",
		Long:  fmsCmdLongDocs,
	}

	fmsCmdLongDocs = `The Field Management System (FMS) provides a number of utilities that are related to the management of one or more fields, and the configuration of a machine to servce as an FMS.  While it may be possible to use any Linux machine as an FMS, we only qualify specific hardware, consult the documentation for more information.`
)

func init() {
	rootCmd.AddCommand(fmsCmd)
}
