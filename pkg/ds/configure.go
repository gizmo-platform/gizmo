package ds

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/vishvananda/netlink"
)

const (
	coresvc = "/etc/runit/core-services/90-gizmo.sh"

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
		"hostapd",
		"dnsmasq",
	}

	return exec.Command("xbps-install", append([]string{"-Suy"}, pkgs...)...).Run()
}

// SetupBoot installs the runtime hooks that startup the configuration
// jobs.
func (ds *DriverStation) SetupBoot() error {
	return ds.doTemplate(coresvc, "tpl/coresvc.sh.tpl", 0644, nil)
}

// Configure installs configuration files into the correct locations
// to permit operation of the network components.  It also restarts
// services as necessary.
func (ds *DriverStation) Configure() error {
	steps := []ConfigureStep{
		ds.configureNetwork,
		ds.configureHostname,
		ds.configureHostAPd,
		ds.configureDHCPCD,
		ds.configureDNSMasq,
		ds.configureGizmo,
		ds.enableServices,
	}
	names := []string{
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
	f, err := os.Create("/etc/hostname")
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "gizmoDS-%d\n", ds.cfg.Team)
	f.Close()

	if err := exec.Command("/usr/bin/hostname", fmt.Sprintf("gizmoDS-%d", ds.cfg.Team)).Run(); err != nil {
		return err
	}

	return nil
}

func (ds *DriverStation) configureHostAPd() error {
	if err := ds.doTemplate(hostAPdConf, "tpl/hostapd.conf.tpl", 0644, ds.cfg); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) configureDHCPCD() error {
	return ds.doTemplate(dhcpcdConf, "tpl/dhcpcd.conf.tpl", 0644, ds.cfg)
}

func (ds *DriverStation) configureDNSMasq() error {
	return ds.doTemplate(dnsmasqConf, "tpl/dnsmasq.conf.tpl", 0644, ds.cfg)
}

func (ds *DriverStation) configureGizmo() error {
	if err := ds.doTemplate(gizmoDSSvc, "tpl/gizmo-ds.run.tpl", 0755, ds.cfg); err != nil {
		return err
	}
	if err := ds.doTemplate(gizmoLinkSvc, "tpl/gizmo-link.run.tpl", 0755, nil); err != nil {
		return err
	}
	if err := ds.doTemplate(gizmoConfigSvc, "tpl/gizmo-config.run.tpl", 0755, nil); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) enableServices() error {
	ds.svc.Enable("hostapd")
	ds.svc.Enable("dnsmasq")
	ds.svc.Enable("gizmo-ds")
	ds.svc.Enable("gizmo-link")
	ds.svc.Enable("gizmo-config")
	return nil
}
