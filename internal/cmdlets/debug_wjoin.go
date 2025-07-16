package cmdlets

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	debugWJoinCmd = &cobra.Command{
		Use:  "wjoin",
		Run:  debugWJoinRun,
		Args: cobra.ExactArgs(1),
	}
)

func init() {
	debugCmd.AddCommand(debugWJoinCmd)
}

func debugWJoinRun(c *cobra.Command, args []string) {
	f, err := os.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening config: %s\n", err)
		return
	}
	defer f.Close()

	cfg := config.GSSConfig{}

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding config: %s\n", err)
		return
	}

	mecode := fmt.Sprintf("WIFI:S:%s;T:WPA;P:%s;H:true;;", cfg.NetSSID, cfg.NetPSK)
	qrterminal.Generate(mecode, qrterminal.L, os.Stdout)
}
