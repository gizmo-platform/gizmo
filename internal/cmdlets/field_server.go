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
	"github.com/spf13/viper"

	"github.com/bestrobotics/gizmo/pkg/metrics"
	"github.com/bestrobotics/gizmo/pkg/gamepad"
	"github.com/bestrobotics/gizmo/pkg/http"
	"github.com/bestrobotics/gizmo/pkg/mqttpusher"
	"github.com/bestrobotics/gizmo/pkg/mqttserver"
	"github.com/bestrobotics/gizmo/pkg/tlm/simple"
)

var (
	fieldServeCmd = &cobra.Command{
		Use:   "serve",
		Short: "serve gamepads to robots",
		Long:  fieldServeCmdLongDocs,
		Run:   fieldServeCmdRun,
	}

	fieldServeCmdLongDocs = `Serve the field.  You must have a
configuration file!`
)

type quad struct {
	Name    string
	Gamepad int
	Pusher  string
}

func init() {
	fieldCmd.AddCommand(fieldServeCmd)
}

func fieldServeCmdRun(c *cobra.Command, args []string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	viper.SetConfigName("config.yml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

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

	err := viper.ReadInConfig()
	if err != nil {
		appLogger.Error("Could not read config.yml", "error", err)
		os.Exit(1)
	}


	stats := metrics.New(metrics.WithLogger(appLogger))
	appLogger.Debug("Stats listeners created")

	jsc := gamepad.NewJSController(gamepad.WithLogger(appLogger))
	quads := []quad{}

	if err := viper.UnmarshalKey("quadrants", &quads); err != nil {
		appLogger.Error("Could not unmarshal fields", "error", err)
		os.Exit(2)
	}

	quadStr := make([]string, len(quads))
	for i, q := range quads {
		if q.Pusher != "self" {
			continue
		}
		jsc.BindController(q.Name, q.Gamepad)
		quadStr[i] = q.Name
	}

	tlm := simple.New(simple.WithLogger(appLogger), simple.WithStartupWG(wg))

	m, err := mqttserver.NewServer(mqttserver.WithLogger(appLogger), mqttserver.WithStartupWG(wg))
	if err != nil {
		appLogger.Error("Error during mqtt initialization", "error", err)
		os.Exit(1)
	}

	p, err := mqttpusher.New(
		mqttpusher.WithLogger(appLogger),
		mqttpusher.WithJSController(&jsc),
		mqttpusher.WithMQTTServer("mqtt://127.0.0.1:1883"),
		mqttpusher.WithStartupWG(wg),
	)
	if err != nil {
		appLogger.Error("Error during mqtt pusher initialization", "error", err)
		quit <- syscall.SIGINT
	}

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithJSController(&jsc),
		http.WithTeamLocationMapper(tlm),
		http.WithPrometheusRegistry(stats.Registry()),
		http.WithQuads(quadStr),
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
		if err := p.Connect(); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
			return
		}
		p.StartLocationPusher()
		p.StartControlPusher()
	}()

	go func() {
		if err := stats.MQTTInit(wg); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	jsc.BeginAutoRefresh(50)
	tlm.Start()

	wg.Wait()
	appLogger.Info("Startup Complete!")

	<-quit
	appLogger.Info("Shutting down...")
	tlm.Stop()
	p.Stop()
	jsc.StopAutoRefresh()
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
