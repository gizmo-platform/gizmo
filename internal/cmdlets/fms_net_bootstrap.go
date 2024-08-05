//go:build linux

package cmdlets

import (
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

var (
	fmsBootstrapNetCmd = &cobra.Command{
		Use:   "bootstrap",
		Short: "bootstrap a new field network immediately after installing OS data",
		Long:  fmsBootstrapNetCmdLongDocs,
		Run:   fmsBootstrapNetCmdRun,
	}

	fmsBootstrapNetCmdLongDocs = `bootstrap-net performs all the first-time setup after using flash-device to install the operating system on your equipment.  This command provides a guided experience that instructs you when to move cables, when to power-cycle devices, and when unrecoverable errors have occured.  The entire process for a 2-field setup should take you about 15 minutes to complete.`
)

func init() {
	fmsNetCmd.AddCommand(fmsBootstrapNetCmd)
	fmsBootstrapNetCmd.Flags().Bool("skip-apply", false, "Skip applying changes")
}

func fmsBootstrapNetCmdRun(c *cobra.Command, args []string) {
	initLogger("bootstrap-net")

	skipApply, _ := c.Flags().GetBool("skip-apply")

	fmsConf, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}
	controller := config.New(
		config.WithFMS(*fmsConf),
		config.WithLogger(appLogger),
		config.WithRouter("100.64.1.1"),
	)

	// Sync with bootstrap state enabled
	if err := controller.SyncState(true); err != nil {
		appLogger.Error("Fatal error synchronizing state", "error", err)
		return
	}

	if skipApply {
		return
	}

	instructions := []string{
		"You are about to complete out of box provisioning for your field.",
		"Prior to this point, you should have used the flash-device command to",
		"install the most recent qualified system image to your scoring box and",
		"field box or boxes.  Begin the process with all devices powered off.",
		"",
		"Connect the scoring table box's second port (the FMS port) directly to",
		"the FMS workstation (this computer).  Connect no other cables or",
		"devices.",
		"",
		"Power on the scoring table box and wait approximately 30 seconds for",
		"it to boot.  Once the device has booted (pattern of lights has",
		"stabilized), confirm this dialog and the scoring table box will be",
		"programmed.  You will receive more instructions on when to connect",
		"field boxes after the main scoring box provisioning completes.",
	}

	for _, line := range instructions {
		fmt.Println(line)
	}

	confirm := func() bool {
		qProceed := &survey.Confirm{
			Message: "Acknowledge and Proceed",
			Default: false,
		}
		proceed := false
		if err := survey.AskOne(qProceed, &proceed); err != nil {
			fmt.Fprintf(os.Stderr, "Impossible error confirming bootstrap: %s\n", err)
		}
		return proceed
	}

	if !confirm() {
		fmt.Println("Bootstrap process aborted!")
		return
	}

	if err := controller.Init(); err != nil {
		appLogger.Error("Fatal error initializing controller", "error", err)
		return
	}

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
	defer func() {
		if err := netlink.LinkDel(bootstrap0); err != nil {
			appLogger.Error("Error removing bootstrap link", "error", err)
		}
	}()

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

	if err := controller.SyncTLM(make(map[int]string)); err != nil {
		appLogger.Error("Could not shim the TLM", "error", err)
		return
	}

	if err := controller.Converge(true, "module.router"); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}
	appLogger.Info("Core network initialization complete, initializing fields")

	instructions = []string{
		"The scoring box has been successfully programmed for your event.",
		"Connect your field boxes to ports 3-5 on the scoring box at this time.",
		"If you are not using a PoE enabled scoring box, connect power to your",
		"field boxes at this time.",
		"",
		"Once connected, wait approximately 30 seconds for your field boxes to",
		"finish booting (pattern of lights has stabilized) and then confirm",
		"this dialog.  You will see some error messages printed as the initial",
		"configuration is programmed, this is normal.",
		"",
		"This process can take up to 10 minutes to complete.",
		"",
	}

	for _, line := range instructions {
		fmt.Println(line)
	}

	if !confirm() {
		fmt.Println("Bootstrap process aborted!")
		return
	}

	for _, field := range fmsConf.Fields {
		provisionFunc := func() error {
			err := controller.Converge(false, fmt.Sprintf("module.field%d", field.ID))
			appLogger.Error("Error configuring field", "field", field.ID, "error", err)
			return err
		}

		bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(time.Minute * 5))
		if err := backoff.Retry(provisionFunc, bo); err != nil {
			appLogger.Error("Permanent error while configuring field", "error", err)
			return
		}
	}

	// Toggle out of bootstrap mode
	if err := controller.SyncState(false); err != nil {
		appLogger.Error("Fatal error synchronizing state", "error", err)
		return
	}

	if err := controller.Converge(false, "module.router"); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}

	for _, field := range fmsConf.Fields {
		provisionFunc := func() error {
			if err := controller.Converge(false, fmt.Sprintf("module.field%d", field.ID)); err != nil {
				appLogger.Error("Error configuring field", "field", field.ID, "error", err)
				return err
			}
			return nil
		}

		bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(time.Minute * 5))
		if err := backoff.Retry(provisionFunc, bo); err != nil {
			appLogger.Error("Permanent error while configuring field", "error", err)
			return
		}
	}

	appLogger.Info("Provisioning Complete")
}
