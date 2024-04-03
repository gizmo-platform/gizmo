package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-sockaddr"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	configCommand = &cobra.Command{
		Use:   "configure",
		Short: "configure prompts for required configuration values",
		Long:  configCmdLongDocs,
		Run:   configCmdRun,
	}

	configCmdLongDocs = `configure prompts in a wizard style for the values that must be manually set.`
)

func init() {
	rootCmd.AddCommand(configCommand)
}

func configCmdRun(c *cobra.Command, args []string) {
	// We throw away this error because the worst case is the
	// default isn't set.
	lAddr, _ := sockaddr.GetPrivateIP()

	qInitial := []*survey.Question{
		{
			Name:     "Team",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Team Number",
			},
		},
		{
			Name:     "UseDriverStation",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Use the driver's station",
				Default: true,
			},
		},
	}

	cfg := config.Config{
		ServerIP: "gizmo-ds",
		NetSSID:  strings.ReplaceAll(uuid.New().String(), "-", ""),
		NetPSK:   strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	if err := survey.Ask(qInitial, &cfg); err != nil {
		fmt.Println(err.Error())
		return
	}

	qAdvanced := []*survey.Question{
		{
			Name:     "UseExtNet",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Use external network controller",
				Default: false,
			},
		},
	}

	if !cfg.UseDriverStation {
		if err := survey.Ask(qAdvanced, &cfg); err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	qExtNet := []*survey.Question{
		{
			Name:     "NetSSID",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Network SSID",
			},
		},
		{
			Name:     "NetPSK",
			Validate: survey.Required,
			Prompt: &survey.Password{
				Message: "Network PSK (Input will be obscured)",
			},
		},
		{
			Name:     "ServerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Address of the driver station (can be an mDNS name)",
				Default: lAddr,
			},
		},
	}

	if cfg.UseExtNet {
		if err := survey.Ask(qExtNet, &cfg); err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	f, err := os.Create("gsscfg.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening settings file: %s\n", err.Error())
		os.Exit(1)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing settings file: %s\n", err.Error())
		os.Exit(1)
	}
}
