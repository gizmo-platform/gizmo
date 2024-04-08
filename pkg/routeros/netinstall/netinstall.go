// Package netinstall wraps the mikrotik routeros netinstall utility
// to permit guided configuration without the need to drive the
// configuration tools directly.  This makes things both more
// approachable and more repeatable.
package netinstall

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	// embed gets imported blank here because we want to do an
	// embedded file, but that just goes into a []byte rather than
	// a full embed.FS
	_ "embed"

	"github.com/hashicorp/go-hclog"
	"github.com/vishvananda/netlink"
)

const (
	// RouterOSVersion contains the most recent qualified version
	// of the suite to be installed.
	RouterOSVersion = "7.14.2"

	// RouterPkg is the most recent qualified firmware package for
	// a root router to be installed to.  In general its safe to
	// use the most recent version, but this is what we tested.
	RouterPkg = "routeros-" + RouterOSVersion + "-mipsbe.npk"

	// FieldPkg is the most recent qualified firmware package for
	// a field device to be installed to.  In general its safe to
	// use the most recent version, but this is what we tested.
	FieldPkg = "wireless-" + RouterOSVersion + "-mipsbe.npk"

	netinstallPkg  = "netinstall-" + RouterOSVersion + ".tar.gz"
	netinstallPath = "/usr/local/bin/netinstall-cli"

	provisionAddr = "192.168.88.2/24"
	targetAddr    = "192.168.88.1"

	// RouterBootstrapNet is the static address and interface that we
	// want to configure as part of the bootstrap configuration.
	RouterBootstrapNet = "/ip/address/add address=169.254.0.1/16 interface=ether2"

	// ImagePath is a location to store routerOS images into.
	ImagePath = "/usr/share/routeros"
)

// Installer wraps functionality associated with installation.
type Installer struct {
	l hclog.Logger

	pkg          string
	bootstrap    string
	bootstrapCtx map[string]string
}

// An InstallerOpt configures an installer
type InstallerOpt func(i *Installer)

// installSteps are phases of an install.
type installStep func() error

//go:embed bootstrap.rsc
var bootstrapCfg string

// WithLogger configures the logging instance for this installer.
func WithLogger(l hclog.Logger) InstallerOpt {
	return func(i *Installer) { i.l = l }
}

// WithPackage configures what package should be installed
func WithPackage(p string) InstallerOpt {
	return func(i *Installer) { i.pkg = p }
}

// WithNetwork configures the network line that is passed to the
// template context
func WithNetwork(n string) InstallerOpt {
	return func(i *Installer) { i.bootstrapCtx["network"] = n }
}

// New returns a new installer configured for use.
func New(opts ...InstallerOpt) *Installer {
	i := new(Installer)
	i.bootstrapCtx = make(map[string]string)
	for _, o := range opts {
		o(i)
	}
	return i
}

// Install runs the installer with a configured set of options
func (i *Installer) Install() error {
	steps := []installStep{
		i.setupNetwork,
		i.makeBootstrap,
		i.doInstall,
	}
	defer i.teardownNetwork()
	defer i.cleanup()

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) makeBootstrap() error {
	f, err := os.CreateTemp("", "*.rsc")
	if err != nil {
		return err
	}
	defer f.Close()
	i.bootstrap = f.Name()
	i.l.Info("Writing configuration to file", "path", i.bootstrap)

	tpl, err := template.New("bootstrap.rsc").Parse(bootstrapCfg)
	if err != nil {
		return err
	}

	if err := tpl.Execute(f, i.bootstrapCtx); err != nil {
		return err
	}
	return nil
}

func (i *Installer) cleanup() error {
	return os.Remove(i.bootstrap)
}

func (i *Installer) doInstall() error {
	args := []string{
		"-s", i.bootstrap,
		"-r", "-a", targetAddr,
		filepath.Join(ImagePath, i.pkg),
	}

	cmd := exec.Command(netinstallPath, args...)

	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		i.l.Info(scanner.Text())
	}
	cmd.Wait()
	return nil
}

func (i *Installer) setupNetwork() error {
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}

	addr, _ := netlink.ParseAddr(provisionAddr)
	if err := netlink.AddrAdd(eth0, addr); err != nil && err.Error() != "file exists" {
		return err
	}

	return nil
}

func (i *Installer) teardownNetwork() error {
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}

	addr, _ := netlink.ParseAddr(provisionAddr)
	if err := netlink.AddrDel(eth0, addr); err != nil {
		return err
	}

	return nil
}
