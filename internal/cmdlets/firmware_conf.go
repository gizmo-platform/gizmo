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

	"github.com/gizmo-platform/gizmo/pkg/firmware"
)

var (
	firmwareConfigCommand = &cobra.Command{
		Use:   "configure",
		Short: "configure prompts for required configuration values",
		Long:  firmwareConfigCmdLongDocs,
		Run:   firmwareConfigCmdRun,
	}

	firmwareConfigCmdLongDocs = `configure prompts in a wizard style for the values that are required to configure the GSS firmware image.`
)

func init() {
	firmwareCmd.AddCommand(firmwareConfigCommand)
}

func firmwareConfigCmdRun(c *cobra.Command, args []string) {
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
			Name:     "UseConsole",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Use the driver's console",
				Default: true,
			},
		},
	}

	cfg := firmware.Config{
		UseAvahi: true,
		ServerIP: "fms.local",
		NetSSID:  strings.ReplaceAll(uuid.New().String(), "-", ""),
		NetPSK:   strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	if err := survey.Ask(qInitial, &cfg); err != nil {
		fmt.Println(err.Error())
		return
	}

	qAdvanced := []*survey.Question{
		{
			Name:     "UseAvahi",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Use Avahi (mDNS)",
				Default: true,
			},
		},
		{
			Name:     "UseExtNet",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Use external network controller",
				Default: false,
			},
		},
	}

	if !cfg.UseConsole {
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
	}

	if cfg.UseExtNet {
		if err := survey.Ask(qExtNet, &cfg); err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	qAvahi := []*survey.Question{
		{
			Name:     "ServerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Address of the field server",
				Default: lAddr,
			},
		},
	}

	if !cfg.UseAvahi {
		if err := survey.Ask(qAvahi, &cfg); err != nil {
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
