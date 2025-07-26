//go:build linux

package cmdlets

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

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

func fmsBootstrapNetCmdRun(c *cobra.Command, args []string) {
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
		rconfig.WithRouter(rconfig.BootstrapAddr),
	)

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

	if err := controller.BootstrapPhase0(); err != nil {
		appLogger.Error("Fatal error during Phase 0", "error", err)
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

	if err := controller.ActivateBootstrapNet(); err != nil {
		appLogger.Error("Fatal error with bootstrap network", "error", err)
		if err := controller.DeactivateBootstrapNet(); err != nil {
			appLogger.Error("Error occured while unbootstrapping network.  You may need to run `ip link del bootstrap0`.", "error", err)
			return
		}
		return
	}

	if err := controller.BootstrapPhase1(); err != nil {
		appLogger.Error("Fatal error during Phase 1 Bootstrap", "error", err)
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

	if err := controller.BootstrapPhase2(); err != nil {
		appLogger.Error("Fatal error during Phase 2", "error", err)
		return
	}

	// Toggle out of bootstrap mode
	if err := controller.BootstrapPhase3(); err != nil {
		appLogger.Error("Fatal error during Phse 3", "error", err)
		return
	}

	appLogger.Info("Provisioning Complete")
}
