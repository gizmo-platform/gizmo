//go:build linux

package cmdlets

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/sysconf"
)

var (
	dsLinkMonitorCmd = &cobra.Command{
		Use:   "linkmon",
		Short: "linkmon restarts dhcpcd if eth0 cycles",
		Run:   dsLinkMonitorCmdRun,
	}
)

func init() {
	dsCmd.AddCommand(dsLinkMonitorCmd)
}

func dsLinkMonitorCmdRun(c *cobra.Command, args []string) {
	initLogger("ds.linkmon")

	linkChanges := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		close(done)
	}()

	if err := netlink.LinkSubscribe(linkChanges, done); err != nil {
		fmt.Fprintf(os.Stderr, "Could not subscribe to link state changes: %s\n", err)
		return
	}

	r := new(sysconf.Runit)
	for {
		select {
		case l := <-linkChanges:
			if l.Attrs().Name == "eth0" {
				appLogger.Info("Operational state change", "state", l.Attrs().OperState)
				switch l.Attrs().OperState {
				case netlink.OperDown:
					r.Restart("gizmo-ds")
				}
			}
		case <-done:
			return
		}
	}
}
