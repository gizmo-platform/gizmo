package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	if firmwareBuildDir == "" {
		firmwareBuildDir, err = os.MkdirTemp("", "gizmo")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create build directory: %s\n", err.Error())
			os.Exit(1)
		}
	}
	if !firmwareBuildDirPreserve {
		defer func() {
			if err := os.RemoveAll(firmwareBuildDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error cleaning up build directory: %s\n", err.Error())
			}
		}()
	}

	fmt.Printf("Extracting firmware to %s\n", firmwareBuildDir)
	if err := firmware.RestoreToDir(firmwareBuildDir); err != nil {
		fmt.Fprintf(os.Stderr, "Could not extract firmware source files: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("Configuring build")
	if err := firmware.ConfigureBuild(firmwareBuildDir, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Could not configure build: %s\n", err.Error())
		os.Exit(1)
	}

	if firmwareBuildExtractOnly {
		os.Exit(0)
	}

	fmt.Println("Building firmware, this may take a few minutes!")
	if err := firmware.Build(firmwareBuildDir); err != nil {
		fmt.Fprintf(os.Stderr, "Could not build: %s\n", err.Error())
		os.Exit(1)
	}

	if firmwareBuildOutputFile == "" {
		cwd, _ := os.Getwd()
		firmwareBuildOutputFile = filepath.Join(cwd, fmt.Sprintf("gss_%d.uf2", cfg.Team))
	}

	fmt.Println("Copying built firmware")
	if err := firmware.CopyFirmware(firmwareBuildDir, firmwareBuildOutputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Could not output build artifact: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Build complete, GSS image: %s\n", firmwareBuildOutputFile)
}
