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
	"github.com/the-maldridge/bestfield/pkg/mqttpusher"
	"github.com/the-maldridge/bestfield/pkg/mqttserver"
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

	err := viper.ReadInConfig()
	if err != nil {
		appLogger.Error("Could not read config.yml", "error", err)
		os.Exit(1)
	}

	prometheusRegistry, prometheusMetrics := stats.NewListener(appLogger)
	appLogger.Debug("Stats listeners created")

	jsc := gamepad.NewJSController(gamepad.WithLogger(appLogger))
	quads := []quad{}

	if err := viper.UnmarshalKey("quadrants", &quads); err != nil {
		appLogger.Error("Could not unmarshal fields", "error", err)
		os.Exit(2)
	}

	quadStr := make([]string, len(quads))
	for i, q := range quads {
		jsc.BindController(q.Name, q.Gamepad)
		quadStr[i] = q.Name
	}

	tlm := shim.TLM{Mapping: make(map[int]string)}

	m, err := mqttserver.NewServer(mqttserver.WithLogger(appLogger))
	if err != nil {
		appLogger.Error("Error during mqtt initialization", "error", err)
		os.Exit(1)
	}

	p, err := mqttpusher.New(
		mqttpusher.WithLogger(appLogger),
		mqttpusher.WithJSController(&jsc),
		mqttpusher.WithTeamLocationMapper(&tlm),
		mqttpusher.WithMQTTServer("mqtt://127.0.0.1:1883"),
	)
	if err != nil {
		appLogger.Error("Error during mqtt pusher initialization", "error", err)
		quit <- syscall.SIGINT
	}

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithJSController(&jsc),
		http.WithTeamLocationMapper(&tlm),
		http.WithPrometheusRegistry(prometheusRegistry),
		http.WithQuads(quadStr),
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
		if err := stats.MqttListen("mqtt://127.0.0.1:1883", prometheusMetrics); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	jsc.BeginAutoRefresh(50)

	<-quit
	appLogger.Info("Shutting down...")
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
