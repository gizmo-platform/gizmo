//go:build linux

package cmdlets

import (
	"context"
	nhttp "net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
	"github.com/gizmo-platform/gizmo/pkg/tlm/net"
)

var (
	fmsRunCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the FMS",
		Long:  fmsRunCmdLongDocs,
		Run:   fmsRunCmdRun,
	}

	fmsRunCmdLongDocs = `The Field Management System (FMS) is at its heart a long-lived server process that services the field, metrics, and all other components of the competition network.  The run command starts that process and leaves it running.  Prior to running this command, you'll need to have run the wizard, flashed all your devices, and bootstrapped the network.  Consult the documentation for more detailed instructions.`
)

func init() {
	fmsCmd.AddCommand(fmsRunCmd)
}

func fmsRunCmdRun(c *cobra.Command, args []string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	initLogger("fms")

	wg := new(sync.WaitGroup)

	fmsConf, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}

	routerAddr := "100.64.0.1"
	controller := config.New(
		config.WithFMS(*fmsConf),
		config.WithLogger(appLogger),
		config.WithRouter(routerAddr),
	)
	appLogger.Debug("Controller Init")

	tlm := net.New(
		net.WithLogger(appLogger),
		net.WithController(controller),
		net.WithSaveState(".tlm.json"),
	)
	if err := tlm.RecoverState(); err != nil {
		appLogger.Warn("Could not recover TLM state", "error", err)
	}
	appLogger.Debug("TLM Init")

	f, err := fms.New(
		fms.WithLogger(appLogger),
		fms.WithTeamLocationMapper(tlm),
		fms.WithFMSConf(*fmsConf),
		fms.WithStartupWG(wg),
	)
	appLogger.Debug("HTTP Init")

	if err != nil {
		appLogger.Error("Error during webserver initialization", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := f.Serve(":8080"); err != nil && err != nhttp.ErrServerClosed {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	wg.Wait()
	appLogger.Info("Startup Complete!")

	<-quit
	appLogger.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f.Shutdown(ctx); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
		os.Exit(2)
	}
}
