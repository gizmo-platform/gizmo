package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
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
	qInitial := []*survey.Question{
		{
			Name:     "Team",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Team Number",
			},
		},
	}

	cfg := config.GSSConfig{
		ServerIP: "gizmo-ds",
		NetSSID:  strings.ReplaceAll(uuid.New().String(), "-", ""),
		NetPSK:   strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
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
