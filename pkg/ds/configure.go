//go:build linux

package ds

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/util"
)

const (
	// We need to be before sysctls, which fire at 08.
	coresvc = "/etc/runit/core-services/06-gizmo.sh"

	sysctlConf     = "/etc/sysctl.conf"
	hostAPdConf    = "/etc/hostapd/hostapd.conf"
	dnsmasqConf    = "/etc/dnsmasq.conf"
	dnsmasqSvc     = "/etc/sv/dnsmasq/run"
	hostapdSvc     = "/etc/sv/hostapd/run"
	gizmoDSSvc     = "/etc/sv/gizmo-ds/run"
	gizmoLinkSvc   = "/etc/sv/gizmo-link/run"
	gizmoConfigSvc = "/etc/sv/gizmo-config/run"
	gizmoLogmonSvc = "/etc/sv/gizmo-logmon/run"
	lldpConf       = "/etc/lldpd.d/gizmo.conf"
)

// Install invokes xbps to install the necessary packages
func (ds *DriverStation) Install() error {
	pkgs := []string{
		"dnsmasq",
		"hostapd",
		"jq",
		"socklog-void",
		"tmux",
		"lldpd",
	}

	return ds.sc.InstallPkgs(pkgs...)
}

// SetupBoot installs the runtime hooks that startup the configuration
// jobs.
func (ds *DriverStation) SetupBoot() error {
	return ds.sc.Template(coresvc, "tpl/coresvc.sh.tpl", 0644, nil)
}

// Configure installs configuration files into the correct locations
// to permit operation of the network components.  It also restarts
// services as necessary.
func (ds *DriverStation) Configure() error {
	steps := []ConfigureStep{
		ds.configureSysctl,
		ds.configureNetwork,
		ds.configureHostname,
		ds.configureHostAPd,
		ds.configureDNSMasq,
		ds.configureGizmo,
		ds.configureLogmon,
		ds.enableServices,
	}
	names := []string{
		"sysctl",
		"network",
		"hostname",
		"hostapd",
		"dnsmasq",
		"gizmo",
		"logmon",
		"enable",
	}

	for i, step := range steps {
		ds.l.Info("Configuring", "step", names[i])
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *DriverStation) configureSysctl() error {
	return ds.sc.Template(sysctlConf, "tpl/sysctl.conf.tpl", 0644, nil)
}

func (ds *DriverStation) configureNetwork() error {
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		ds.l.Error("Could not retrieve ethernet link", "error", err)
		return err
	}

	err = netlink.LinkAdd(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		Name:         "br0",
		HardwareAddr: util.NumberToMAC(ds.cfg.Team, 1),
	}})
	if err != nil && err.Error() != "file exists" {
		ds.l.Error("Could not create bridge device", "error", err)
		return err
	}

	br0, err := netlink.LinkByName("br0")
	if err != nil {
		ds.l.Error("Could not get handle to bridge interface", "error", err)
		return err
	}

	if err := netlink.LinkSetMaster(eth0, br0); err != nil {
		ds.l.Error("Error adding link to bridge", "error", err)
		return err
	}

	if err := netlink.LinkSetUp(eth0); err != nil {
		ds.l.Error("Error enabling eth0", "error", err)
		return err
	}

	if err := netlink.LinkSetUp(br0); err != nil {
		ds.l.Error("Error enabling br0", "error", err)
		return err
	}

	addr, _ := netlink.ParseAddr(fmt.Sprintf("10.%d.%d.2/24", int(ds.cfg.Team/100), ds.cfg.Team%100))
	if err := netlink.AddrAdd(br0, addr); err != nil {
		ds.l.Error("Error adding address to br0", "error", err)
		return err
	}

	_, subnet, _ := net.ParseCIDR("100.64.0.0/24")
	gw := net.IPv4(10, byte(ds.cfg.Team/100), byte(ds.cfg.Team%100), 1)
	if err := netlink.RouteAdd(&netlink.Route{Dst: subnet, Gw: gw}); err != nil {
		ds.l.Error("Error adding FMS route", "error", err)
		return err
	}

	return nil
}

func (ds *DriverStation) configureHostname() error {
	name := fmt.Sprintf("gizmoDS-%d", ds.cfg.Team)
	if err := ds.sc.Template(lldpConf, "tpl/lldp.conf.tpl", 0644, name); err != nil {
		return err
	}
	return ds.sc.SetHostname(name)
}

func (ds *DriverStation) configureHostAPd() error {
	ctx := map[string]string{
		"NetSSID": ds.cfg.NetSSID,
		"NetPSK":  ds.cfg.NetPSK,
		"Channel": "1",
	}

	if err := ds.sc.Template(hostAPdConf, "tpl/hostapd.conf.tpl", 0644, ctx); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) configureDNSMasq() error {
	return ds.sc.Template(dnsmasqConf, "tpl/dnsmasq.conf.tpl", 0644, ds.cfg)
}

func (ds *DriverStation) configureGizmo() error {
	if err := ds.sc.Template(gizmoDSSvc, "tpl/gizmo-ds.run.tpl", 0755, ds.cfg); err != nil {
		return err
	}
	if err := ds.sc.Template(gizmoLinkSvc, "tpl/gizmo-link.run.tpl", 0755, nil); err != nil {
		return err
	}
	if err := ds.sc.Template(gizmoConfigSvc, "tpl/gizmo-config.run.tpl", 0755, nil); err != nil {
		return err
	}

	svcs := []string{gizmoDSSvc, gizmoLinkSvc, gizmoConfigSvc, dnsmasqSvc, hostapdSvc}
	for _, svc := range svcs {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(svc), "log"), 0755); err != nil {
			return err
		}

		if err := os.Link("/usr/bin/vlogger", filepath.Join(filepath.Dir(svc), "log", "run")); err != nil && !os.IsExist(err) {
			return err
		}
	}

	return nil
}

func (ds *DriverStation) configureLogmon() error {
	return ds.sc.Template(gizmoLogmonSvc, "tpl/gizmo-logmon.run.tpl", 0755, nil)
}

func (ds *DriverStation) enableServices() error {
	ds.sc.Enable("dnsmasq")
	ds.sc.Enable("gizmo-config")
	ds.sc.Enable("gizmo-ds")
	ds.sc.Enable("gizmo-logmon")
	ds.sc.Enable("gizmo-link")
	ds.sc.Enable("hostapd")
	ds.sc.Enable("socklog-unix")
	ds.sc.Enable("nanoklogd")
	ds.sc.Enable("lldpd")
	for _, i := range []int{1, 4, 5, 6} {
		ds.sc.Disable(fmt.Sprintf("agetty-tty%d", i))
	}
	return nil
}
