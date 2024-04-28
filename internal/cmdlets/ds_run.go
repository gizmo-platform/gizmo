//go:build linux

package cmdlets

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/ds"
)

var (
	dsRunCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the main driver-station process",
		Long:  dsRunCmdLongDocs,
		Run:   dsRunCmdRun,
		Args:  cobra.ExactArgs(1),
	}

	dsRunCmdLongDocs = `The driver's station is a long lived process that either provides local control functionality or provides input to a centralized field management system (FMS).  This command handles the process of providing those services and dynamically switching between them.`
)

func init() {
	dsCmd.AddCommand(dsRunCmd)
}

func dsRunCmdRun(c *cobra.Command, args []string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "driver-station",
		Level: hclog.LevelFromString(ll),
	})

	cfg, err := config.Load(args[0])
	if err != nil {
		appLogger.Error("Error loading config", "error", err)
		return
	}

	drv := ds.New(ds.WithLogger(appLogger), ds.WithGSSConfig(*cfg))

	go drv.Run()
	<-quit
	appLogger.Info("Shutdown requested")
	drv.Stop()
}
