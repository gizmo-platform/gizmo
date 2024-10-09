//go:build linux

package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/ds"
)

var (
	dsConfigureCmd = &cobra.Command{
		Use:   "configure",
		Short: "configure reads the gsscfg.json and configures the driver's station",
		Long:  dsConfigureCmdLongDocs,
		Run:   dsConfigureCmdRun,
		Args:  cobra.ExactArgs(1),
	}

	dsConfigureCmdLongDocs = `configure reads the gsscfg.json file and uses it to configure the operating system.  This expects that prerequisite installation has been completed previously.`
)

func init() {
	dsCmd.AddCommand(dsConfigureCmd)
}

func dsConfigureCmdRun(c *cobra.Command, args []string) {
	initLogger("ds")

	f, err := os.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening config: %s\n", err)
		return
	}
	defer f.Close()

	cfg := config.Config{}

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding config: %s\n", err)
		return
	}

	d := ds.New(ds.WithGSSConfig(cfg), ds.WithLogger(appLogger))

	if err := d.Configure(); err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring: %s\n", err)
		return
	}
}
