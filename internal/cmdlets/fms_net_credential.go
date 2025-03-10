//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
)

var (
	fmsNetCredentialsCmd = &cobra.Command{
		Use:       "credential <admin|view|auto>",
		Short:     "Print the specified credential to stdout (useful with xsel)",
		Run:       fmsNetCredentialCmdRun,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"admin", "view", "auto"},
	}
)

func init() {
	fmsNetCmd.AddCommand(fmsNetCredentialsCmd)
}

func fmsNetCredentialCmdRun(c *cobra.Command, args []string) {
	fmsConf, err := fms.NewConfig(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %s\n", err.Error())
		return
	}

	switch args[0] {
	case "admin":
		fmt.Println(fmsConf.AdminPass)
	case "view":
		fmt.Println(fmsConf.ViewPass)
	case "auto":
		fmt.Println(fmsConf.AutoPass)
	}
}
