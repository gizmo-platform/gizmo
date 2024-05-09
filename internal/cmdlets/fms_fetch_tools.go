//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
)

var (
	fmsFetchToolsCmd = &cobra.Command{
		Use:   "fetch-tools",
		Short: "download tools to disk",
		Long:  fmsFetchToolsCmdLongDocs,
		Run:   fmsFetchToolsCmdRun,
	}

	fmsFetchToolsCmdLongDocs = `fetch-tools downloads qualified tools from the vendor website for field network components.  Internet is required to run this command.`
)

func init() {
	fmsCmd.AddCommand(fmsFetchToolsCmd)
}

func fmsFetchToolsCmdRun(c *cobra.Command, args []string) {
	initLogger("fetch-tools")

	if err := netinstall.FetchTools(appLogger); err != nil {
		appLogger.Error("Unable to fetch one or more tools, see above", "error", err)
	}
}
