package cmdlets

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

var (
	fmsFlashRouterCmd = &cobra.Command{
		Use:   "flash-router",
		Short: "provides guided installation instructions to configure a field router",
		Long:  fmsFlashRouterCmdLongDocs,
		Run:   fieldHardwareFlashRouterCmdRun,
	}

	fmsFlashRouterCmdLongDocs = `flash-router provides a guided experience to install the operating system on the field router.  You can use this command to return the field to a clean state after a competition, or to update software on a new field router`
)

func init() {
	fmsCmd.AddCommand(fmsFlashRouterCmd)
}

func fieldHardwareFlashRouterCmdRun(c *cobra.Command, args []string) {
	instructions := []string{
		"Welcome to the field-flash utility.",
		"",
		"This process will guide you through the process of installing the most",
		"recently confirmed working firmware on your field router.",
		"",
		"Before you begin, you should ensure that you have the field router, an",
		"unfolded paperclip or similar instrument, and a cable to connect port",
		"1 of the field router (says 'Internet'), and the FMS workstation (this",
		"computer).",
		"",
		"After you confirm this message, hold down the reset button using the",
		"paperclip and connect power.  You may remove the paperclip after you",
		"see a message containing the phrase 'client'.",
	}

	for _, line := range instructions {
		fmt.Println(line)
	}

	qProceed := []*survey.Question{{
		Name:     "Confirm",
		Validate: survey.Required,
		Prompt: &survey.Confirm{
			Message: "I have completed the above setup steps",
			Default: false,
		},
	}}

	proceed := struct {
		Confirm bool
	}{}

	if err := survey.Ask(qProceed, &proceed); err != nil {
		fmt.Fprintf(os.Stderr, "Impossible error confirming flash: %s\n", err)
		return
	}

	if !proceed.Confirm {
		fmt.Println("Flash process aborted!")
		return
	}

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "flash-router",
		Level: hclog.LevelFromString(ll),
	})

	installer := netinstall.New(
		netinstall.WithLogger(appLogger),
		netinstall.WithPackage(netinstall.RouterPkg),
		netinstall.WithNetwork(netinstall.RouterBootstrapNet),
	)

	if err := installer.Install(); err != nil {
		appLogger.Error("Fatal error during install", "error", err)
		return
	}

	appLogger.Info("Flashing complete, you may now disconnect cables.")
}
