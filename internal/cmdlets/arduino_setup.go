package cmdlets

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const (
	arduinoCoreURL = "https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json"
)

var (
	arduinoSetupCmd = &cobra.Command{
		Use:   "setup",
		Short: "setup installs the pico core and gizmo library",
		Long:  arduinoSetupCmdLongDocs,
		Run:   arduinoSetupCmdRun,
	}

	arduinoSetupCmdLongDocs = `setup installs the Arduino-Pico core as well as installing the Gizmo library.`
)

func init() {
	arduinoCmd.AddCommand(arduinoSetupCmd)
}

// This is really the wrong way to do this on a lot of levels.  What
// we should be doing here is working out where the arduino-cli is
// located and then running `arduino-cli daemon` which then can be
// interacted with over gRPC like a real program.  That's a future
// enhancement of this code and the firmware utilities, but this was
// faster to implement and get out the door in a working build.
func arduinoSetupCmdRun(c *cobra.Command, args []string) {
	arduino, err := exec.LookPath("arduino-cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "arduino-cli is not installed!")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Using arduino-cli at %s\n", arduino)

	boardSetupArgs := []string{
		"--additional-urls", arduinoCoreURL,
		"core", "install", "rp2040:rp2040",
	}

	fmt.Fprint(os.Stdout, "Attempting to install Pico Core\n")
	output, err := exec.Command(arduino, boardSetupArgs...).CombinedOutput()
	fmt.Fprint(os.Stdout, string(output))
	fmt.Fprint(os.Stdout, "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not install the Pico core.  Check the output above\n")
		os.Exit(2)
	}

	fmt.Fprint(os.Stdout, "Attempting to refresh library index\n")
	output, err = exec.Command(arduino, []string{"lib", "update-index"}...).CombinedOutput()
	fmt.Fprint(os.Stdout, string(output))
	fmt.Fprint(os.Stdout, "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not update library index.  Check the output above\n")
		os.Exit(2)
	}

	libInstallArgs := []string{"lib", "install", "Gizmo"}
	fmt.Fprint(os.Stdout, "Attempting to install Gizmo Libraries\n")
	output, err = exec.Command(arduino, libInstallArgs...).CombinedOutput()
	fmt.Fprint(os.Stdout, string(output))
	fmt.Fprint(os.Stdout, "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not install the Gizmo Library.  Check the output above\n")
		os.Exit(2)
	}
}
