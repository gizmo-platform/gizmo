package sysconf

import (
	"fmt"
	"os"
	"os/exec"
)

// SetHostname sets up the hostname both in the hostname file, and by
// changing the immediate machine hostname.
func (sc *SysConf) SetHostname(name string) error {
	f, err := os.Create("/etc/hostname")
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "%s\n", name)
	f.Close()

	if err := exec.Command("/usr/bin/hostname", name).Run(); err != nil {
		return err
	}

	return nil
}
