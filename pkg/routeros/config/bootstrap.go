package config

import (
	"os/exec"
)

const (
	// BootstrapAddr points to where the scoring router would be
	// during the bootstrap scenario.
	BootstrapAddr = "100.64.1.1"
)

// BootstrapPhase0 sets up the intial structures on disk and gets to a
// point where further assumptions around bootstrapping can proceed.
func (c *Configurator) BootstrapPhase0() error {
	c.ctx["RouterBootstrap"] = true
	c.ctx["FieldBootstrap"] = true

	// Sync with bootstrap state enabled
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		return err
	}

	if err := c.SyncTLM(make(map[int]string)); err != nil {
		c.l.Error("Could not shim the TLM", "error", err)
		return err
	}
	return nil
}

// BootstrapPhase1 handles the bring up of the core router to a point
// where it is providing DHCP and the rest of the system is able to
// communicate.
func (c *Configurator) BootstrapPhase1() error {
	if err := c.waitForROS(BootstrapAddr, c.fc.AutoUser, c.fc.AutoPass); err != nil {
		c.l.Error("ROS is not available", "error", err)
		return err
	}

	if err := c.Converge(true, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		return err
	}
	c.ctx["RouterBootstrap"] = false
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error syncing state", "error", err)
		return err
	}
	if err := c.Converge(false, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		return err
	}
	if err := c.DeactivateBootstrapNet(); err != nil {
		c.l.Warn("Could not unbootstrap the local network", "error", err)
	}
	if err := exec.Command("dhcpcd", "--rebind", "eth0").Run(); err != nil {
		c.l.Warn("Could not rebind dhcpcd, you probably don't have an address!", "error", err)
	}
	if err := c.waitForFMSIP(); err != nil {
		c.l.Error("Did not aquire FMS IP, cannot continue!", "error", err)
		return err
	}
	return nil
}

// BootstrapPhase2 handles the bootstrapping of fields.
func (c *Configurator) BootstrapPhase2() error {
	c.ctx["FieldBootstrap"] = true

	// Sync with bootstrap state enabled
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		return err
	}

	if err := c.convergeFields(); err != nil {
		c.l.Error("Error converging fields", "error", err)
		return err
	}
	return nil
}

// BootstrapPhase3 toggles out of bootstrap mode, and returns the
// system to its normal operating state.
func (c *Configurator) BootstrapPhase3() error {
	c.ctx["FieldBootstrap"] = false
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		return err
	}

	if err := c.Converge(false, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		return err
	}

	if err := c.convergeFields(); err != nil {
		c.l.Error("Error converging fields", "error", err)
		return err
	}

	return nil
}
