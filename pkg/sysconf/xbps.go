package sysconf

import (
	"os/exec"
)

// InstallPkgs interfaces with the system package manager to install
// OS packages.
func (sc *SysConf) InstallPkgs(pkgs ...string) error {
	return exec.Command("xbps-install", append([]string{"-Suy"}, pkgs...)...).Run()

}
