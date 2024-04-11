package cmdlets

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

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
	fmsReconcileNetCmd.Flags().Bool("bootstrap", false, "Enable bootstrap mode")
}

func fmsReconcileNetCmdRun(c *cobra.Command, args []string) {
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
	bootstrap, _ := c.Flags().GetBool("bootstrap")
	if bootstrap {
		routerAddr = "100.64.1.1"
		fmsAddr := "100.64.1.2"
		appLogger.Info("Bootstrap mode enabled")

		eth0, err := netlink.LinkByName("eth0")
		if err != nil {
			appLogger.Error("Could not retrieve ethernet link", "error", err)
			return
		}

		bootstrap0 := &netlink.Vlan{
			LinkAttrs:    netlink.LinkAttrs{Name: "bootstrap0", ParentIndex: eth0.Attrs().Index},
			VlanId:       2,
			VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
		}

		if err := netlink.LinkAdd(bootstrap0); err != nil && err.Error() != "file exists" {
			appLogger.Error("Could not create bootstrapping interface", "error", err)
			return
		}

		for _, int := range []netlink.Link{eth0, bootstrap0} {
			if err := netlink.LinkSetUp(int); err != nil {
				appLogger.Error("Error enabling eth0", "error", err)
				return
			}
		}

		addr, _ := netlink.ParseAddr(fmsAddr + "/24")
		if err := netlink.AddrAdd(bootstrap0, addr); err != nil {
			appLogger.Error("Could not add IP", "error", err)
			return
		}
	}

	controller := config.New(
		config.WithFMS(*fmsConf),
		config.WithLogger(appLogger),
		config.WithRouter(routerAddr),
	)

	if err := controller.SyncState(); err != nil {
		appLogger.Error("Fatal error synchronizing state", "error", err)
		return
	}

	if err := controller.Converge(bootstrap); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}

	if bootstrap {
		bootstrap0, _ := netlink.LinkByName("bootstrap0")
		if err := netlink.LinkDel(bootstrap0); err != nil {
			appLogger.Error("Error removing bootstrap link", "error", err)
			return
		}
	}
}
