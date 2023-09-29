package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-sockaddr"
	"github.com/spf13/cobra"
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
			Name:     "ServerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Address of the field server",
				Default: lAddr,
			},
		},
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

	cfg := struct {
		ServerIP string
		NetSSID  string
		NetPSK   string
	}{}

	if err := survey.Ask(qInitial, &cfg); err != nil {
		fmt.Println(err.Error())
		return
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
