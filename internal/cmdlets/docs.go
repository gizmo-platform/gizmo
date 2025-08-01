package cmdlets

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/docs"
)

var (
	docsCmd = &cobra.Command{
		Use:   "docs",
		Short: "docs serves the documentation locally on port 8080",
		Run:   docsCmdRun,
	}
)

func init() {
	rootCmd.AddCommand(docsCmd)
}

func docsCmdRun(c *cobra.Command, args []string) {
	mux := http.NewServeMux()
	mux.Handle("/", docs.MakeHandler("/"))
	srv := new(http.Server)
	srv.Addr = ":8080"
	srv.Handler = mux

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Error binding docs server, do you already have something running?\n%s\n", err)
			return
		}
	}()
	fmt.Println("Documentation available on port 8080, open your browser to http://localhost:8080/")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	fmt.Println("Goodbye!")
	srv.Shutdown(context.Background())
}
