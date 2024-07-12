//go:build linux

package cmdlets

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/ds"
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
	initLogger("ds")

	linkChanges := make(chan netlink.LinkUpdate)
	addrChanges := make(chan netlink.AddrUpdate)
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

	if err := netlink.AddrSubscribe(addrChanges, done); err != nil {
		fmt.Fprintf(os.Stderr, "Could not subscribe to addr state changes: %s\n", err)
		return
	}

	var prevState netlink.LinkOperState
	r := new(ds.Runit)
	for {
		select {
		case l := <-linkChanges:
			if l.Attrs().Name == "eth0" {
				if l.Attrs().OperState != prevState {
					if err := r.Restart("dhcpcd"); err != nil {
						appLogger.Warn("Error restarting dhcpcd", "error", err)
					}
					appLogger.Info("Restarted dhcpcd")
				}
				prevState = l.Attrs().OperState
			}
		case a := <-addrChanges:
			if a.NewAddr {
				if err := r.Restart("gizmo-ds"); err != nil {
					appLogger.Warn("Error restarting gizmo-ds", "error", err)
				}
				appLogger.Info("Restarted gizmo-ds")
			}
		case <-done:
			return
		}
	}
}
