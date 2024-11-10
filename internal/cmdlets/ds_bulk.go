//go:build linux

package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	dsBulkCmd = &cobra.Command{
		Use:   "bulk-configure",
		Short: "Generate gsscfg.json files in bulk",
		Long:  dsBulkCmdLongDocs,
		Run:   dsBulkCmdRun,
	}

	dsBulkCmdLongDocs = `bulk-configure generates gsscfg.json files in bulk by consuming the same CSV files that the FMS itself uses to generate all the team mappings.`
)

func init() {
	dsCmd.AddCommand(dsBulkCmd)
}

func dsBulkCmdRun(c *cobra.Command, args []string) {
	answers := struct {
		Base  int
		Count int
	}{}

	prompts := []*survey.Question{
		{
			Name:     "Base",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Base Team Number",
			},
		},
		{
			Name:     "Count",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Total number of teams",
			},
		},
	}

	if err := survey.Ask(prompts, &answers); err != nil {
		fmt.Fprintf(os.Stderr, "Error asking questions %s\n", err)
		return
	}

	if err := os.MkdirAll("configs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
	}
	for i := answers.Base; i <= answers.Base+answers.Count; i++ {
		cfg := config.Config{
			Team:             i,
			NetSSID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
			NetPSK:           strings.ReplaceAll(uuid.New().String(), "-", ""),
			ServerIP:         "ds.gizmo",
		}
		td := filepath.Join("configs", fmt.Sprintf("team%d", i))
		if err := os.MkdirAll(td, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			return
		}
		cf, err := os.Create(filepath.Join(td, "gsscfg.json"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file: %s\n", err)
			return
		}
		defer cf.Close()
		cf.Chmod(0644)

		if err := json.NewEncoder(cf).Encode(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
			return
		}
	}
}
