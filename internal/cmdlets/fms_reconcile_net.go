package cmdlets

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

var (
	fmsReconcileNetCmd = &cobra.Command{
		Use:   "reconcile-net",
		Short: "compare existing network and desired configuration, reconciling differnces.",
		Long:  fmsReconcileNetCmdLongDocs,
		Run:   fmsReconcileNetCmdRun,
	}

	fmsReconcileNetCmdLongDocs = `reconcile-net performs a comparison between the existing state of a field network element and the desired state and attempts to close the gap.  This is normally done for you between every state change when the field is remapped, but you can also run it on demand in the event of a power failure or field fault.`
)

func init() {
	fmsCmd.AddCommand(fmsReconcileNetCmd)
	fmsReconcileNetCmd.Flags().Bool("skip-apply", false, "Skip applying changes")
	fmsReconcileNetCmd.Flags().Bool("skip-refresh", false, "Skip refreshing current state")
}

func fmsReconcileNetCmdRun(c *cobra.Command, args []string) {
	skipApply, _ := c.Flags().GetBool("skip-apply")
	skipRefresh, _ := c.Flags().GetBool("skip-refresh")
	skipRefresh = !skipRefresh

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "flash-router",
		Level: hclog.LevelFromString(ll),
	})

	fmsConf, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}
	routerAddr := "100.64.0.1"
	controller := config.New(
		config.WithFMS(*fmsConf),
		config.WithLogger(appLogger),
		config.WithRouter(routerAddr),
	)

	// Not in bootstrap mode, pass a "false" here.
	if err := controller.SyncState(false); err != nil {
		appLogger.Error("Fatal error synchronizing state", "error", err)
		return
	}

	if skipApply {
		return
	}

	if err := controller.Converge(skipRefresh, ""); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}
}
