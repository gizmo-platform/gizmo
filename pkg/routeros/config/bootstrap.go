package config

import (
	"os/exec"
)

// BootstrapPhase0 sets up the intial structures on disk and gets to a
// point where further assumptions around bootstrapping can proceed.
func (c *Configurator) BootstrapPhase0() error {
	c.ctx["RouterBootstrap"] = true
	c.ctx["FieldBootstrap"] = true

	// Sync with bootstrap state enabled
	c.es.PublishLogLine("[LOG] => Synchronizing state files")
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		c.es.PublishError(err)
		return err
	}

	if err := c.SyncTLM(make(map[int]string)); err != nil {
		c.l.Error("Could not shim the TLM", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[ACTION] => Plug the FMS workstation into Scoring Box port #2 and turn it on")
	c.es.PublishLogLine("[CLICK] => You may now click on Phase 1")
	c.es.PublishActionComplete("Network Bootstrap (Phase 0)")
	return nil
}

// BootstrapPhase1 handles the bring up of the core router to a point
// where it is providing DHCP and the rest of the system is able to
// communicate.
func (c *Configurator) BootstrapPhase1() error {
	c.es.PublishLogLine("[LOG] => Activating bootstrap network configuration")
	if err := c.ActivateBootstrapNet(); err != nil {
		c.l.Error("Error activating bootstrap net", "error", err)
		c.es.PublishError(err)
		if err := c.DeactivateBootstrapNet(); err != nil {
			c.l.Warn("Could not unbootstrap the local network", "error", err)
			c.es.PublishError(err)
		}
		return err
	}
	c.es.PublishLogLine("[LOG] => Waiting for connection to Scoring Box")
	if err := c.waitForROS(BootstrapAddr, c.fc.AutoUser, c.fc.AutoPass); err != nil {
		c.l.Error("ROS is not available", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.ctx["RouterBootstrap"] = true
	c.routerAddr = BootstrapAddr
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Error synchronizing state", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => Applying configuration to Scoring Box")
	if err := c.Converge(true, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.ctx["RouterBootstrap"] = false
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error syncing state", "error", err)
		c.es.PublishError(err)
		return err
	}
	if err := c.Converge(false, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => Finished configuring Scoring Box")
	c.es.PublishLogLine("[LOG] => Deactivating bootstrap network configuration")
	if err := c.DeactivateBootstrapNet(); err != nil {
		c.l.Warn("Could not unbootstrap the local network", "error", err)
		c.es.PublishError(err)
		return err
	}
	if err := exec.Command("dhcpcd", "--rebind", "eth0").Run(); err != nil {
		c.l.Warn("Could not rebind dhcpcd, you probably don't have an address!", "error", err)
		c.es.PublishError(err)
	}
	c.routerAddr = NormalAddr
	c.es.PublishLogLine("[LOG] => Waiting for FMS Workstatation to acquire network")
	if err := c.waitForFMSIP(); err != nil {
		c.l.Error("Did not aquire FMS IP, cannot continue!", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => FMS Workstation has acquried network connection")
	c.es.PublishLogLine("[ACTION] => Plug field(s) into port(s) 3-8 and turn them on")
	c.es.PublishLogLine("[CLICK] => You may now click on Phase 2")
	c.es.PublishActionComplete("Network Bootstrap (Phase 1)")
	return nil
}

// BootstrapPhase2 handles the bootstrapping of fields.
func (c *Configurator) BootstrapPhase2() error {
	c.ctx["FieldBootstrap"] = true

	// Sync with bootstrap state enabled
	c.es.PublishLogLine("[LOG] => Synchronizing state files")
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		c.es.PublishError(err)
		return err
	}

	c.es.PublishLogLine("[LOG] => Configuring Fields")
	if err := c.convergeFields(); err != nil {
		c.l.Error("Error converging fields", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => Finished configuring fields")
	c.es.PublishLogLine("[CLICK] => You may now click on Phase 3")
	c.es.PublishActionComplete("Network Bootstrap (Phase 2)")
	return nil
}

// BootstrapPhase3 toggles out of bootstrap mode, and returns the
// system to its normal operating state.
func (c *Configurator) BootstrapPhase3() error {
	c.ctx["FieldBootstrap"] = false
	c.es.PublishLogLine("[LOG] => Synchronizing state files")
	if err := c.SyncState(c.ctx); err != nil {
		c.l.Error("Fatal error synchronizing state", "error", err)
		c.es.PublishError(err)
		return err
	}

	c.es.PublishLogLine("[LOG] => Transitioning Scoring Box to normal mode")
	if err := c.Converge(false, "module.router"); err != nil {
		c.l.Error("Fatal error converging state", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => Finished transitioning Scoring Box to normal mode")

	c.es.PublishLogLine("[LOG] => Transitioning Fields to normal mode")
	if err := c.convergeFields(); err != nil {
		c.l.Error("Error converging fields", "error", err)
		c.es.PublishError(err)
		return err
	}
	c.es.PublishLogLine("[LOG] => Finished transitioning Fields to normal mode")
	c.es.PublishLogLine("[LOG] => Bootstrap Complete")
	c.es.PublishActionComplete("Network Bootstrap (Phase 3)")
	return nil
}
