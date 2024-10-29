package cmdlets

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run:   versionCmdRun,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func versionCmdRun(c *cobra.Command, args []string) {
	fmt.Println("Gizmo Platform Tools")
	fmt.Printf("Version: %s\n", buildinfo.Version)
	fmt.Printf("Commit: %s\n", buildinfo.Commit)
	fmt.Printf("Built: %s\n", buildinfo.BuildDate)
}
