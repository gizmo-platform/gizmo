package cmdlets

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/firmware"
)

var (
	firmwareBulkCmd = &cobra.Command{
		Use:   "bulk <teamlist>",
		Short: "bulk produces a collection of uf2 files based on a list of teams",
		Run:   firmwareBulkCmdRun,
		Args:  cobra.ExactArgs(1),
	}
)

func init() {
	firmwareCmd.AddCommand(firmwareBulkCmd)
}

func firmwareBulkCmdRun(c *cobra.Command, args []string) {
	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "field",
		Level: hclog.LevelFromString(ll),
	})
	appLogger.Info("Log level", "level", appLogger.GetLevel())

	cfgFile, err := os.Open("gsscfg.json")
	if err != nil {
		appLogger.Error("Error loading gsscfg.json, have you run gizmo firmware configure?", "error", err.Error())
		os.Exit(1)
	}
	defer cfgFile.Close()

	cfg := firmware.Config{}
	if err := json.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		appLogger.Error("Error loading config file", "error", err.Error())
		os.Exit(1)
	}

	tcsv, err := os.Open(args[0])
	if err != nil {
		appLogger.Error("Error opening team CSV", "error", err)
		os.Exit(1)
	}
	defer tcsv.Close()

	teams := []map[string]string{}
	var header []string
	r := csv.NewReader(tcsv)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			appLogger.Error("Error decoding CSV", "error", err)
			os.Exit(2)
		}
		if header == nil {
			header = record
			for col := range header {
				header[col] = strings.ReplaceAll(header[col], "Team Name", "Name")
				header[col] = strings.ReplaceAll(header[col], "Team Number", "Number")
				header[col] = strings.ReplaceAll(header[col], "Hub Name", "Hub")
			}
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			teams = append(teams, dict)
		}
	}

	opts := []firmware.BuildOption{
		firmware.WithGSSConfig(cfg),
		firmware.WithEphemeralBuildDir(),
	}

	f := firmware.NewFactory(appLogger)

	for _, team := range teams {
		num, err := strconv.Atoi(team["Number"])
		if err != nil {
			appLogger.Warn("Bad team number", "team", team["Name"], "hub", team["Hub"], "number", team["Number"], "error", err)
			return
		}
		output := fmt.Sprintf("gss_%d.uf2", num)
		if err := f.Build(append(opts, firmware.WithTeamNumber(num), firmware.WithBuildOutputFile(output))...); err != nil {
			appLogger.Error("Build failed", "error", err)
			os.Exit(1)
		}
	}
}
