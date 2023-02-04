package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
	"github.com/the-maldridge/bestfield/pkg/http"
)

type tlm struct{}

func (tlm *tlm) GetFieldForTeam(team int) (string, error) { return "field1::red", nil }

func (tlm *tlm) SetScheduleStep(_ int) error { return nil }

func main() {
	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "field",
		Level: hclog.LevelFromString(ll),
	})

	jsc := gamepad.NewJSController(gamepad.WithLogger(appLogger))
	jsc.BindController("field1::red", 0)

	w, err := http.NewServer(
		http.WithLogger(appLogger),
		http.WithJSController(&jsc),
		http.WithTeamLocationMapper(&tlm{}),
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
}
