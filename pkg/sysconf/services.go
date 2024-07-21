package sysconf

import (
	"os"
	"os/exec"
	"path/filepath"
)

const (
	serviceDir = "/etc/sv"
	runsvDir   = "/etc/runit/runsvdir/default"
)

// Runit provides a handle to the service supervisor calls
type Runit struct{}

// Enable configures a service to start at boot and starts it.
func (r *Runit) Enable(svc string) error {
	return os.Symlink(filepath.Join(serviceDir, svc), filepath.Join(runsvDir, svc))
}

// Disable removes a service from the boot set.
func (r *Runit) Disable(svc string) error {
	return os.Remove(filepath.Join(runsvDir, svc))
}

// Start requests runit to immediately start a service.
func (r *Runit) Start(svc string) error {
	return exec.Command("sv", "up", svc).Run()
}

// Stop requests runit to immediately stop a service.
func (r *Runit) Stop(svc string) error {
	return exec.Command("sv", "down", svc).Run()
}

// Restart requests runit to signal the service to stop, then starts
// it again.
func (r *Runit) Restart(svc string) error {
	return exec.Command("sv", "restart", svc).Run()
}
