package cmdlets

import (
	"context"
	nhttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/http"
	"github.com/gizmo-platform/gizmo/pkg/mdns"
	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/mqttpusher"
	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
	"github.com/gizmo-platform/gizmo/pkg/tlm/simple"
)

var (
	fieldPracticeCmd = &cobra.Command{
		Use:   "practice",
		Short: "practice <team>",
		Long:  fieldPracticeCmdLongDocs,
		Run:   fieldPracticeCmdRun,
		Args:  cobra.ExactArgs(1),
	}

	fieldPracticeCmdLongDocs = `Practice sets up a field server that only has one quadrant called "PRACTICE" and that only expects one gamepad to be available.  This enables a team to practice without running an entire field.`
)

func init() {
	fieldCmd.AddCommand(fieldPracticeCmd)
}

func fieldPracticeCmdRun(c *cobra.Command, args []string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "field",
		Level: hclog.LevelFromString(ll),
	})
	appLogger.Info("Log level", "level", appLogger.GetLevel())
	wg := new(sync.WaitGroup)

	stats := metrics.New(metrics.WithLogger(appLogger))
	appLogger.Debug("Stats listeners created")

	tlm := simple.New(
		simple.WithLogger(appLogger),
		simple.WithStartupWG(wg),
		simple.WithMetrics(stats),
	)

	m, err := mqttserver.NewServer(
		mqttserver.WithLogger(appLogger),
		mqttserver.WithStartupWG(wg),
	)
	if err != nil {
		appLogger.Error("Error during mqtt initialization", "error", err)
		os.Exit(1)
	}

	p, err := mqttpusher.New(
		mqttpusher.WithLogger(appLogger),
		mqttpusher.WithMQTTServer("mqtt://127.0.0.1:1883"),
		mqttpusher.WithStartupWG(wg),
		mqttpusher.WithQuadMap(map[string]int{"field1:practice": 0}),
	)
	if err != nil {
		appLogger.Error("Error during mqtt pusher initialization", "error", err)
		quit <- syscall.SIGINT
	}

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithTeamLocationMapper(tlm),
		http.WithPrometheusRegistry(stats.Registry()),
		http.WithQuads([]string{"field1:practice"}),
		http.WithStartupWG(wg),
	)
	if err != nil {
		appLogger.Error("Error initializing http", "error", err)
		quit <- syscall.SIGINT
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
		if err := p.Connect(); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
			return
		}
	}()

	go func() {
		if err := stats.MQTTInit(wg); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	tNum, err := strconv.Atoi(args[0])
	if err != nil {
		appLogger.Error("Team number must be a number", "error", err)
		quit <- syscall.SIGINT
	}

	tlm.InsertOnDemandMap(map[int]string{tNum: "field1:practice"})
	tlm.Start()
	stats.StartFlusher()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := mdns.NewServer(args[0]); err != nil {
			appLogger.Error("Could not publish mdns service", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	wg.Wait()
	appLogger.Info("Startup Complete!")

	<-quit
	appLogger.Info("Shutting down...")
	tlm.Stop()
	p.Stop()
	stats.Shutdown()
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
