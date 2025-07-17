//go:build linux

package cmdlets

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
)

var (
	fmsFetchPackagesCmd = &cobra.Command{
		Use:   "fetch-packages",
		Short: "download packages to disk for later installation",
		Long:  fmsFetchPackagesCmdLongDocs,
		Run:   fmsFetchPackagesCmdRun,
	}

	fmsFetchPackagesCmdLongDocs = `fetch-packages downloads qualified firmware images from the vendor website for the field router and the field radio.  Internet is required to run this command.`
)

func init() {
	fmsSetupCmd.AddCommand(fmsFetchPackagesCmd)
}

func fmsFetchPackagesCmdRun(c *cobra.Command, args []string) {
	initLogger("fetch-packages")

	f := netinstall.NewFetcher(
		netinstall.WithFetcherLogger(appLogger),
		netinstall.WithFetcherPackageDir(os.Getenv("GIZMO_ROS_IMAGE_PATH")),
	)
	if err := f.FetchPackages(); err != nil {
		appLogger.Error("Unable to fetch one or more packages, see above", "error", err)
	}
}
