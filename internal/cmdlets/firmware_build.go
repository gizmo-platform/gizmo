package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/bestrobotics/gizmo/pkg/firmware"
)

var (
	firmwareBuildCmd = &cobra.Command{
		Use:   "build",
		Short: "build produces a u2f firmware file",
		Run:   firmwareBuildCmdRun,
	}

	firmwareBuildDir         string
	firmwareBuildDirPreserve bool
	firmwareBuildExtractOnly bool
	firmwareBuildOutputFile  string
)

func init() {
	firmwareCmd.AddCommand(firmwareBuildCmd)
	firmwareBuildCmd.Flags().StringVar(&firmwareBuildDir, "directory", "", "Directory for a build to take place in (tmpdir if unset)")
	firmwareBuildCmd.Flags().BoolVar(&firmwareBuildDirPreserve, "preserve", false, "Retain build directory")
	firmwareBuildCmd.Flags().BoolVar(&firmwareBuildExtractOnly, "extract-only", false, "Don't compile, just extract")
	firmwareBuildCmd.Flags().StringVar(&firmwareBuildOutputFile, "output", "", "File to output build to")
}

func firmwareBuildCmdRun(c *cobra.Command, args []string) {
	cfgFile, err := os.Open("gsscfg.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading gsscfg.json, have you run gizmo firmware configure? (%s)\n", err.Error())
		os.Exit(1)
	}
	defer cfgFile.Close()

	cfg := firmware.Config{}
	if err := json.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config file: %s\n", err.Error())
		os.Exit(1)
	}

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "field",
		Level: hclog.LevelFromString(ll),
	})
	appLogger.Info("Log level", "level", appLogger.GetLevel())

	opts := []firmware.BuildOption{
		firmware.WithGSSConfig(cfg),
		firmware.WithBuildOutputFile(firmwareBuildOutputFile),
	}

	if firmwareBuildDir == "" {
		opts = append(opts, firmware.WithEphemeralBuildDir())
	} else {
		opts = append(opts, firmware.WithBuildDir(firmwareBuildDir))
	}

	if firmwareBuildExtractOnly {
		opts = append(opts, firmware.WithKeepBuildDir())
		opts = append(opts, firmware.WithExtractOnly())
	}

	f := firmware.NewFactory(appLogger)
	if err := f.Build(opts...); err != nil {
		appLogger.Error("Build failed", "error", err)
		os.Exit(1)
	}
}
