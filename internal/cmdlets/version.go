package cmdlets

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
	fmt.Printf("Built: %s\n", BuildDate)
}
