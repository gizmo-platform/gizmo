//go:build linux

package cmdlets

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/fms"
)

var (
	fmsConfigServerCmd = &cobra.Command{
		Use:   "config-server",
		Short: "configure a selection of gizmos as they are plugged in",
		Long:  fmsConfigServerCmdLongDocs,
		Run:   fmsConfigServerCmdRun,
	}

	fmsConfigServerCmdLongDocs = `config-server provides a means of configuring a large number of Gizmo devices by prompting for what config out of the FMS configuration should be loaded to the device.`
)

func init() {
	fmsCmd.AddCommand(fmsConfigServerCmd)
	fmsConfigServerCmd.Flags().Bool("oneshot", false, "Only apply config to one Gizmo")
}

func fmsConfigServerCmdRun(c *cobra.Command, args []string) {
	oneshot, _ := c.Flags().GetBool("oneshot")
	initLogger("config-server")

	fmsConf, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}

	teams := make(map[string]config.Config)
	names := []string{}
	numbers := make(map[int]string)
	for id, team := range fmsConf.Teams {
		t := config.Config{
			Team:    id,
			NetSSID: team.SSID,
			NetPSK:  team.PSK,
		}
		teams[team.Name] = t
		numbers[t.Team] = team.Name
		names = append(names, team.Name)
	}

	prvdr := func(t int) config.Config {
		prompt := &survey.Select{
			Message: "Select configuration to bind to this Gizmo",
			Options: names,
			Default: numbers[t],
		}
		selected := ""
		if err := survey.AskOne(prompt, &selected); err != nil {
			return config.Config{}
		}
		return teams[selected]
	}

	opts := []config.Option{config.WithProvider(prvdr), config.WithLogger(appLogger)}
	if oneshot {
		opts = append(opts, config.WithOneshotMode())
	}
	srv := config.NewServer(opts...)

	if err := srv.Serve(); err != nil {
		appLogger.Error("Error initializing config server", "error", err)
	}
}
