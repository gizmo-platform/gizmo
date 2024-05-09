// Package cmdlets contains the main entrypoints of the various
// functions that the gizmo tool can perform.
package cmdlets

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

var (
	// Version is the release number for this build
	Version = "dev"

	// Commit is the specific git hash
	Commit = "UNKNOWN"

	// BuildDate is the build timestamp
	BuildDate = time.Now().String()

	rootCmd = &cobra.Command{
		Use:   "gizmo",
		Short: "Entrypoint for all Gizmo commands",
		Long:  rootCmdLongDocs,
	}
	rootCmdLongDocs = `The Gizmo Platform provides servers for field control, configuration for your joysticks, and tools to program the system processor on your robot control board.`

	appLogger = hclog.NewNullLogger()
)

// Entrypoint is the entrypoint into all cmdlets, it will dispatch to
// the right one.
func Entrypoint() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func initLogger(name string) {
	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger = hclog.New(&hclog.LoggerOptions{
		Name:  name,
		Level: hclog.LevelFromString(ll),
	})
	appLogger.Info("Log level", "level", appLogger.GetLevel())
}
