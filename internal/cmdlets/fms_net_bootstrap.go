//go:build linux

package cmdlets

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/config"
	rconfig "github.com/gizmo-platform/gizmo/pkg/routeros/config"
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
	fmsBootstrapNetCmd.Flags().Bool("skip-init", false, "Skip terraform initialization")
	fmsBootstrapNetCmd.Flags().Bool("init-only", false, "Only perform terraform initialization and exit")
}

const (
	bootstrapAddr = "100.64.1.1"
)

func fmsBootstrapNetCmdRun(c *cobra.Command, args []string) {
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

	bootstrapNet := func() error {
		// Setup the bootstrap mode which talks via a distinct vlan to
		// configure everything.  This is necessary since part of the
		// setup changes the layer 2 network and we need to not change
		// the network we're configuring from.
		fmsAddr := "100.64.1.2"
		appLogger.Info("Bootstrap mode enabled")

		eth0, err := netlink.LinkByName("eth0")
		if err != nil {
			appLogger.Error("Could not retrieve ethernet link", "error", err)
			return err
		}

		bootstrap0 := &netlink.Vlan{
			LinkAttrs:    netlink.LinkAttrs{Name: "bootstrap0", ParentIndex: eth0.Attrs().Index},
			VlanId:       2,
			VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
		}

		if err := netlink.LinkAdd(bootstrap0); err != nil && err.Error() != "file exists" {
			appLogger.Error("Could not create bootstrapping interface", "error", err)
			return err
		}

		for _, int := range []netlink.Link{eth0, bootstrap0} {
			if err := netlink.LinkSetUp(int); err != nil {
				appLogger.Error("Error enabling eth0", "error", err)
				return err
			}
		}

		addr, _ := netlink.ParseAddr(fmsAddr + "/24")
		if err := netlink.AddrAdd(bootstrap0, addr); err != nil {
			appLogger.Error("Could not add IP", "error", err)
			return err
		}
		return nil
	}

	unbootstrapNet := func() error {
		bootstrap0, err := netlink.LinkByName("bootstrap0")
		if err != nil {
			appLogger.Error("Could not retrieve ethernet link", "error", err)
			return err
		}

		if err := netlink.LinkDel(bootstrap0); err != nil {
			appLogger.Error("Error removing bootstrap link", "error", err)
			return err
		}
		return nil
	}

	waitForROS := func(wg *sync.WaitGroup, addr, user, pass string) {
		defer wg.Done()
		cl := http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		req := &http.Request{
			Method: http.MethodGet,
			URL: &url.URL{
				Scheme: "http",
				Host:   addr,
				Path:   "/rest/system/identity",
				User:   url.UserPassword(user, pass),
			},
		}

		retryFunc := func() error {
			_, err := cl.Do(req)
			if err != nil {
				appLogger.Info("Waiting for device", "address", addr, "error", err)
			}
			return err
		}

		if err := backoff.Retry(retryFunc, backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Second*30))); err != nil {
			appLogger.Error("Permanent error encountered while waiting for RouterOS", "error", err)
			appLogger.Error("You need to reboot network boxes and try again")
			return
		}
	}

	waitForFMSIP := func() error {
		appLogger.Debug("Aquiring eth0")
		eth0, err := netlink.LinkByName("eth0")
		if err != nil {
			appLogger.Error("Could not retrieve ethernet link", "error", err)
			return err
		}

		fmsIP, _ := netlink.ParseAddr("100.64.0.2/24")

		retryFunc := func() error {
			appLogger.Debug("Requesting addresses from eth0")
			addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
			if err != nil {
				appLogger.Error("Error listing addresses", "error", err)
				return err
			}

			for _, a := range addrs {
				appLogger.Debug("Checking IP", "have", a.String(), "want", fmsIP.String())
				if a.Equal(*fmsIP) {
					return nil
				}
			}

			return errors.New("No FMS IP")
		}
		appLogger.Debug("Link acquired, waiting for address")

		if err := backoff.Retry(retryFunc, backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Second*30))); err != nil {
			appLogger.Error("Permanent error encountered while waiting for dhcp address", "error", err)
			appLogger.Error("You may be able to recover from this by restarting dhcpcd")
		}
		return nil
	}

	initLogger("bootstrap-net")

	skipApply, _ := c.Flags().GetBool("skip-apply")
	skipInit, _ := c.Flags().GetBool("skip-init")
	initOnly, _ := c.Flags().GetBool("init-only")

	fmsConf, err := config.NewFMSConfig(appLogger)
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}
	controller := rconfig.New(
		rconfig.WithFMS(fmsConf),
		rconfig.WithLogger(appLogger),
		rconfig.WithRouter(bootstrapAddr),
	)
	ctx := make(map[string]interface{})
	ctx["RouterBootstrap"] = true
	ctx["FieldBootstrap"] = true

	// Sync with bootstrap state enabled
	if err := controller.SyncState(ctx); err != nil {
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
		"Power on the scoring table box and wait approximately 2 minutes for",
		"it to boot.  Once the device has booted (pattern of lights has",
		"stabilized), confirm this dialog and the scoring table box will be",
		"programmed.  You will receive more instructions on when to connect",
		"field boxes after the main scoring box provisioning completes.",
	}
	if !initOnly {
		for _, line := range instructions {
			fmt.Println(line)
		}
		if !confirm() {
			fmt.Println("Bootstrap process aborted!")
			return
		}
	}

	if !skipInit {
		if err := controller.Init(); err != nil {
			appLogger.Error("Fatal error initializing controller", "error", err)
			return
		}
	}

	if initOnly {
		return
	}

	if err := bootstrapNet(); err != nil {
		appLogger.Error("Fatal error with bootstrap network", "error", err)
		if err := unbootstrapNet(); err != nil {
			appLogger.Error("Error occured while unbootstrapping network.  You may need to run `ip link del bootstrap0`.", "error", err)
			return
		}
		return
	}

	// At this point we're ready to actually configure the root
	// router.  This requires the TLM to sync even with empty
	// state since that results in state files on disk.
	if err := controller.SyncTLM(make(map[int]string)); err != nil {
		appLogger.Error("Could not shim the TLM", "error", err)
		return
	}

	var swg sync.WaitGroup
	swg.Add(1)
	go waitForROS(&swg, bootstrapAddr, fmsConf.AutoUser, fmsConf.AutoPass)
	swg.Wait()

	// We limit module.router here to configure only the scoring
	// router.  This needs to get configured first since this sets
	// up the DHCP reservations for the field access points.
	// Without setting this up we wouldn't be able to assert the
	// location of the field devices later.
	if err := controller.Converge(true, "module.router"); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}
	ctx["RouterBootstrap"] = false
	if err := controller.SyncState(ctx); err != nil {
		appLogger.Error("Fatal error syncing state", "error", err)
		return
	}
	if err := controller.Converge(false, "module.router"); err != nil {
		appLogger.Error("Fatal error converging state", "error", err)
		return
	}
	if err := unbootstrapNet(); err != nil {
		appLogger.Warn("Could not unbootstrap the local network", "error", err)
	}
	if err := exec.Command("dhcpcd", "--rebind", "eth0").Run(); err != nil {
		appLogger.Warn("Could not rebind dhcpcd, you probably don't have an address!", "error", err)
	}
	if err := waitForFMSIP(); err != nil {
		appLogger.Error("Did not aquire FMS IP, cannot continue!", "error", err)
		return
	}

	appLogger.Info("Core network initialization complete, initializing fields")

	instructions = []string{
		"The scoring box has been successfully programmed for your event.",
		"Connect your field boxes to ports 3-5 on the scoring box at this time.",
		"If you are not using a PoE enabled scoring box, connect power to your",
		"field boxes at this time.",
		"",
		"Once connected, wait approximately 2 minutes for your field boxes to",
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
		swg.Add(1)
		go waitForROS(&swg, field.IP, fmsConf.AutoUser, fmsConf.AutoPass)
		swg.Wait()
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

	// Toggle out of bootstrap mode
	ctx["FieldBootstrap"] = false
	if err := controller.SyncState(ctx); err != nil {
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
