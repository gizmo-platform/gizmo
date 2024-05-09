//go:build linux

package cmdlets

import (
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	dsConfigServerCmd = &cobra.Command{
		Use:   "config-server <file>",
		Short: "config-server provides configuration data to an attached gizmo",
		Long:  dsConfigServerCmdLongDocs,
		Run:   dsConfigServerCmdRun,
		Args:  cobra.ExactArgs(1),
	}

	dsConfigServerCmdLongDocs = `config-server provides a means of a gizmo to receive the gsscfg.json file.  It does this by listening to the requested serial port and then providing the configuration file once a magic handshake string has been received.  Consult the documentation for further information about how this handshake process works, and if you need to drive it manually how to do that.`
)

func init() {
	dsCmd.AddCommand(dsConfigServerCmd)
}

func dsConfigServerCmdRun(c *cobra.Command, args []string) {
	initLogger("config-server")

	cfg, err := config.Load(args[0])
	if err != nil {
		appLogger.Error("Error loading config", "error", err)
		return
	}

	prvdr := func() config.Config {
		return *cfg
	}

	srv := config.NewServer(config.WithProvider(prvdr), config.WithLogger(appLogger))

	if err := srv.Serve(); err != nil {
		appLogger.Error("Error initializing config server", "error", err)
	}
}
