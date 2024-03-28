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
	changes := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		close(done)
	}()

	if err := netlink.LinkSubscribe(changes, done); err != nil {
		fmt.Fprintf(os.Stderr, "Could not subscribe to link state changes: %s\n", err)
		return
	}

	var prevState netlink.LinkOperState
	r := new(ds.Runit)
	for {
		select {
		case l := <-changes:
			if l.Attrs().Name == "eth0" {
				if l.Attrs().OperState != prevState {
					r.Restart("dhcpcd")
				}
				prevState = l.Attrs().OperState
			}
		case <-done:
			return
		}
	}
}
