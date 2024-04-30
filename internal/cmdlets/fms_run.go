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

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/http"
	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
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

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "fms",
		Level: hclog.LevelFromString(ll),
	})
	appLogger.Info("Log level", "level", appLogger.GetLevel())
	wg := new(sync.WaitGroup)

	fmsConf, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}

	stats := metrics.New(metrics.WithLogger(appLogger))
	appLogger.Debug("Stats listeners created")

	routerAddr := "100.64.0.1"
	controller := config.New(
		config.WithFMS(*fmsConf),
		config.WithLogger(appLogger),
		config.WithRouter(routerAddr),
	)

	tlm := net.New(
		net.WithLogger(appLogger),
		net.WithMetrics(stats),
		net.WithController(controller),
		net.WithStartupWG(wg),
	)

	m, err := mqttserver.NewServer(mqttserver.WithLogger(appLogger), mqttserver.WithStartupWG(wg))
	if err != nil {
		appLogger.Error("Error during mqtt initialization", "error", err)
		os.Exit(1)
	}

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithTeamLocationMapper(tlm),
		http.WithPrometheusRegistry(stats.Registry()),
		http.WithFMSConf(*fmsConf),
		http.WithMQTTServer(m),
		http.WithStartupWG(wg),
	)

	if err != nil {
		appLogger.Error("Error during webserver initialization", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := m.Serve(":1883"); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	go func() {
		if err := w.Serve(":8080"); err != nil && err != nhttp.ErrServerClosed {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	go func() {
		if err := stats.MQTTInit(wg); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	stats.StartFlusher()
	tlm.Start()

	wg.Wait()
	appLogger.Info("Startup Complete!")

	<-quit
	appLogger.Info("Shutting down...")
	stats.Shutdown()
	appLogger.Info("Stats Stopped")
	tlm.Stop()
	appLogger.Info("TLM Stopped")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.Shutdown(ctx); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
		os.Exit(2)
	}
	if err := m.Shutdown(); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
		os.Exit(2)
	}
}
