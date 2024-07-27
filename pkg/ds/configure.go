//go:build linux

package ds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vishvananda/netlink"
)

const (
	// We need to be before sysctls, which fire at 08.
	coresvc = "/etc/runit/core-services/06-gizmo.sh"

	sysctlConf     = "/etc/sysctl.conf"
	hostAPdConf    = "/etc/hostapd/hostapd.conf"
	dnsmasqConf    = "/etc/dnsmasq.conf"
	dhcpcdConf     = "/etc/dhcpcd.conf"
	gizmoDSSvc     = "/etc/sv/gizmo-ds/run"
	gizmoLinkSvc   = "/etc/sv/gizmo-link/run"
	gizmoConfigSvc = "/etc/sv/gizmo-config/run"
)

// Install invokes xbps to install the necessary packages
func (ds *DriverStation) Install() error {
	pkgs := []string{
		"dnsmasq",
		"hostapd",
		"mqttcli",
		"tmux",
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
		ds.configureDHCPCD,
		ds.configureDNSMasq,
		ds.configureGizmo,
		ds.enableServices,
	}
	names := []string{
		"sysctl",
		"network",
		"hostname",
		"hostapd",
		"dhcpcd",
		"dnsmasq",
		"gizmo",
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

	err = netlink.LinkAdd(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: "br0"}})
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

	return nil
}

func (ds *DriverStation) configureHostname() error {
	return ds.sc.SetHostname(fmt.Sprintf("gizmoDS-%d", ds.cfg.Team))
}

func (ds *DriverStation) configureHostAPd() error {
	if err := ds.sc.Template(hostAPdConf, "tpl/hostapd.conf.tpl", 0644, ds.cfg); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) configureDHCPCD() error {
	return ds.sc.Template(dhcpcdConf, "tpl/dhcpcd.conf.tpl", 0644, ds.cfg)
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

	for _, svc := range []string{gizmoDSSvc, gizmoLinkSvc, gizmoConfigSvc} {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(svc), "log"), 0755); err != nil {
			return err
		}

		if err := os.Link("/usr/bin/vlogger", filepath.Join(filepath.Dir(svc), "log", "run")); err != nil {
			return err
		}
	}

	return nil
}

func (ds *DriverStation) enableServices() error {
	ds.sc.Enable("hostapd")
	ds.sc.Enable("dnsmasq")
	ds.sc.Enable("gizmo-ds")
	ds.sc.Enable("gizmo-link")
	ds.sc.Enable("gizmo-config")
	return nil
}
