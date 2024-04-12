package cmdlets

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/gizmo-platform/gizmo/pkg/fms"
	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
)

var (
	fmsFlashDeviceCmd = &cobra.Command{
		Use:   "flash-device",
		Short: "provides guided installation instructions to configure a field device",
		Long:  fmsFlashDeviceCmdLongDocs,
		Run:   fieldHardwareFlashDeviceCmdRun,
	}

	fmsFlashDeviceCmdLongDocs = `flash-device provides a guided experience to install the operating system on the field device.  You can use this command to return the field to a clean state after a competition, or to update software on a new field device`
)

func init() {
	fmsCmd.AddCommand(fmsFlashDeviceCmd)
}

func fieldHardwareFlashDeviceCmdRun(c *cobra.Command, args []string) {
	instructions := []string{
		"Welcome to the field-flash utility.",
		"",
		"This process will guide you through the process of installing the most",
		"recently confirmed working firmware on your field device.",
		"",
		"Before you begin, you should ensure that you have the field device, an",
		"unfolded paperclip or similar instrument, and a cable to connect port",
		"1 of the field device (says 'Internet'), and the FMS workstation (this",
		"computer).",
	}

	confirmPrompt := []string{
		"After you confirm this message, hold down the reset button using the",
		"paperclip and connect power.  You may remove the paperclip after you",
		"see a message containing the phrase 'client'.",
		"",
		"Ready to proceed",
	}

	for _, line := range instructions {
		fmt.Println(line)
	}

	qDevice := &survey.Select{
		Message: "Select the type of device you are flashing",
		Options: []string{"Field Box", "Scoring Table Box"},
		Default: "Scoring Table Box",
	}
	fDev := ""
	if err := survey.AskOne(qDevice, &fDev); err != nil {
		fmt.Fprintf(os.Stderr, "Impossible error confirming flash: %s\n", err)
		return
	}

	qProceed := &survey.Confirm{
		Message: strings.Join(confirmPrompt, "\n"),
		Default: false,
	}
	proceed := false
	if err := survey.AskOne(qProceed, &proceed); err != nil {
		fmt.Fprintf(os.Stderr, "Impossible error confirming flash: %s\n", err)
		return
	}

	if !proceed {
		fmt.Println("Flash process aborted!")
		return
	}

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "flash-device",
		Level: hclog.LevelFromString(ll),
	})

	cfg, err := fms.LoadConfig("fms.json")
	if err != nil {
		appLogger.Error("Could not load fms.json, have you run the wizard yet?", "error", err)
		return
	}

	pkg := netinstall.RouterPkg
	if fDev == "Scoring Table Box" {
		pkg = netinstall.FieldPkg
	}

	installer := netinstall.New(
		netinstall.WithLogger(appLogger),
		netinstall.WithPackage(pkg),
		netinstall.WithFMS(cfg),
	)

	if err := installer.Install(); err != nil {
		appLogger.Error("Fatal error during install", "error", err)
		return
	}

	appLogger.Info("Flashing complete, you may now disconnect cables.")
}
