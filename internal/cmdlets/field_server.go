package cmdlets

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
	"github.com/the-maldridge/bestfield/pkg/http"
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

	err := viper.ReadInConfig()
	if err != nil {
		appLogger.Error("Could not read config.yml", "error", err)
		os.Exit(1)
	}

	jsc := gamepad.NewJSController(gamepad.WithLogger(appLogger))

	quads := []quad{}
	if err := viper.UnmarshalKey("quadrants", &quads); err != nil {
		appLogger.Error("Could not unmarshal fields", "error", err)
		os.Exit(2)
	}

	for _, q := range quads {
		jsc.BindController(q.Name, q.Gamepad)
	}

	appLogger.Info("fields", "fields", quads)

	jsc.BeginAutoRefresh(50)
	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithJSController(&jsc),
		http.WithTeamLocationMapper(&shim.TLM{Mapping: map[int]string{1234: "field1:red"}}),
	)

	if err != nil {
		log.Println("Error during webserver initialization", err.Error())
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := w.Serve(":8080"); err != nil {
			appLogger.Error("Error initializing", "error", err)
			quit <- syscall.SIGINT
		}
	}()

	<-quit
	appLogger.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.Shutdown(ctx); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
		os.Exit(2)
	}
	jsc.StopAutoRefresh()
}
