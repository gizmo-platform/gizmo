package cmdlets

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

var (
	dsConfigServerCmd = &cobra.Command{
		Use:   "config-server <file>",
		Short: "config-server provides configuration data to an attached gizmo",
		Long:  dsConfigServerCmdLongDocs,
		Run:   dsConfigServerCmdRun,
		Args:  cobra.ExactArgs(1),
	}

	dsConfigServerCmdLongDocs = `config-server provides a means of a gizmo to receive the gsscfg.json file.  It does this by listening to the requested serial port and then providing the configuration file once a magic handshake string has been received.  Consult the documentation for further information about how this handshake process works, and if you need to drive it manually how to do that.`
)

func init() {
	dsCmd.AddCommand(dsConfigServerCmd)
}

func dsConfigServerCmdRun(c *cobra.Command, args []string) {
	t := time.NewTicker(time.Second)
	pname := ""
out:
	for range t.C {
		ports, err := enumerator.GetDetailedPortsList()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
		for _, port := range ports {
			fmt.Println("Found port", port.Name)
			if !port.IsUSB {
				// We know the Gizmo must be connected via USB
				continue
			}
			if port.VID == "2e8a" && port.PID == "f00a" {
				pname = port.Name
				break out
			}
		}
	}
	t.Stop()

	cfg, err := os.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open config file: %s\n", err)
		return
	}
	defer cfg.Close()

	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(pname, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open port: %s\n", err)
		return
	}
	defer port.Close()
	if err := port.SetReadTimeout(time.Second * 15); err != nil {
		return
	}

	fmt.Println("Waiting for Gizmo")
	scanner := bufio.NewScanner(bufio.NewReader(port))
	for scanner.Scan() {
		if scanner.Text() == "GIZMO_REQUEST_CONFIG" {
			break
		}
	}

	fmt.Println("Uploading config")
	io.Copy(port, cfg)
	if err := port.Drain(); err != nil {
		fmt.Fprintf(os.Stderr, "Error draining port: %s\n", err)
		return
	}
	fmt.Println("Upload complete")
}
