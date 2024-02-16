package cmdlets

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	arduinoMSIURL = "https://downloads.arduino.cc/arduino-ide/arduino-ide_latest_Windows_64bit.msi"
)

var (
	arduinoInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Attempt to install the Arduino IDE",
		Long:  arduinoInstallCmdLongDocs,
		Run:   arduinoInstallCmdRun,
	}

	arduinoInstallCmdLongDocs = `install attempts to fetch the latest version of the Arduino IDE and install it.  You may need to run this as an administrator.`
)

func init() {
	arduinoCmd.AddCommand(arduinoInstallCmd)
}

func arduinoInstallCmdRun(c *cobra.Command, args []string) {
	os.Exit(func() int {
		d, err := os.MkdirTemp("", "gizmo")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create temporary directory: %s\n", err)
			return 2
		}
		defer os.RemoveAll(d)

		f, err := os.Create(filepath.Join(d, "arduino.msi"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create arduino.msi temporary file: %s\n", err)
			return 2
		}
		defer f.Close()

		fmt.Fprintf(os.Stdout, "Fetching Arduino Installer to %s\n", f.Name())
		resp, err := http.Get(arduinoMSIURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not fetch Arduino installer: %s\n", err)
			return 2
		}
		defer resp.Body.Close()
		if _, err = io.Copy(f, resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "Error storing Arduino installer: %s\n", err)
			return 2
		}
		f.Close()

		fmt.Fprint(os.Stdout, "Launching Arduino Installer\n")
		cmd := exec.Command("msiexec", "/i", "arduino.msi")
		cmd.Dir = d
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not launch Arduino installer: %s\n", err)
			return 2
		}
		fmt.Fprint(os.Stdout, string(output))
		return 0
	}())
}
