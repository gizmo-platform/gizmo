package cmdlets

import (
	"context"
	nhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/the-maldridge/bestfield/internal/stats"
	"github.com/the-maldridge/bestfield/pkg/gamepad"
	"github.com/the-maldridge/bestfield/pkg/http"
	"github.com/the-maldridge/bestfield/pkg/mqtt"
	"github.com/the-maldridge/bestfield/pkg/tlm/shim"
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
}

func init() {
	fieldCmd.AddCommand(fieldServeCmd)
}

func fieldServeCmdRun(c *cobra.Command, args []string) {
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

	err := viper.ReadInConfig()
	if err != nil {
		appLogger.Error("Could not read config.yml", "error", err)
		os.Exit(1)
	}

	prometheusRegistry, prometheusMetrics := stats.NewStatsListener(appLogger)
	appLogger.Debug("Stats listeners created")

	jsc := gamepad.NewJSController(gamepad.WithLogger(appLogger))
	quads := []quad{}

	if err := viper.UnmarshalKey("quadrants", &quads); err != nil {
		appLogger.Error("Could not unmarshal fields", "error", err)
		os.Exit(2)
	}

	for _, q := range quads {
		jsc.BindController(q.Name, q.Gamepad)
	}
	jsc.BeginAutoRefresh(50)

	tlm := shim.TLM{Mapping: make(map[int]string)}

	m, err := mqtt.NewServer(
		mqtt.WithLogger(appLogger),
		mqtt.WithJSController(&jsc),
		mqtt.WithTeamLocationMapper(&tlm),
	)

	if err != nil {
		appLogger.Error("Error during mqtt initialization", "error", err)
		os.Exit(1)
	}

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithJSController(&jsc),
		http.WithTeamLocationMapper(&tlm),
		http.WithPrometheusRegistry(prometheusRegistry),
	)

	if err != nil {
		appLogger.Error("Error during webserver initialization", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

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

	if err := stats.MqttListen("mqtt://127.0.0.1:1883", prometheusMetrics); err != nil {
		appLogger.Error("Error initializing", "error", err)
		quit <- syscall.SIGINT
	}

	m.StartControlPusher()
	<-quit
	appLogger.Info("Shutting down...")
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
	m.StopControlPusher()
	jsc.StopAutoRefresh()
}
