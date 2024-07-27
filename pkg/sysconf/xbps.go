package sysconf

import (
	"bufio"
	"io"
	"os/exec"
)

// InstallPkgs interfaces with the system package manager to install
// OS packages.
func (sc *SysConf) InstallPkgs(pkgs ...string) error {
	cmd := exec.Command("xbps-install", append([]string{"-Suy"}, pkgs...)...)

	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe

	cmd.Start()
	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			sc.l.Info(scanner.Text())
		}
	}()

	return cmd.Wait()
}
