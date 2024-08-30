//go:build linux

package cmdlets

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

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
	addrChanges := make(chan netlink.AddrUpdate)
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	var prevAddr net.IP

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

	r := new(sysconf.Runit)
	delayRestart := func(svc string) {
		// This delay ensures that linkstate has settled prior
		// to bouncing services.  The kernel fires the events
		// for link changes as they occur, so if we
		// immediately do things, the device state may not be
		// settled.  The correct way to handle this is to do a
		// more intelligent poll and verify approach to make
		// sure things are done changing, but this is easier
		// to implement and understand.
		time.Sleep(time.Second * 2)
		if err := r.Restart(svc); err != nil {
			appLogger.Warn("Error restarting service", "service", svc, "error", err)
		}
		appLogger.Info("Restarted Service", "service", svc)
	}

	var prevState netlink.LinkOperState
	for {
		select {
		case l := <-linkChanges:
			if l.Attrs().Name == "eth0" {
				if l.Attrs().OperState != prevState {
					go delayRestart("dhcpcd")
				}
				prevState = l.Attrs().OperState
			}
		case a := <-addrChanges:
			if a.NewAddr && !slices.Equal(a.LinkAddress.IP, prevAddr) {
				prevAddr = a.LinkAddress.IP
				appLogger.Info("New Address", "address", a.LinkAddress.IP)
				go delayRestart("gizmo-ds")
			}
		case <-done:
			return
		}
	}
}
