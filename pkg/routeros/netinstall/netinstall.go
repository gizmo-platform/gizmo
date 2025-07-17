// Package netinstall wraps the mikrotik routeros netinstall utility
// to permit guided configuration without the need to drive the
// configuration tools directly.  This makes things both more
// approachable and more repeatable.
package netinstall

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	// embed gets imported blank here because we want to do an
	// embedded file, but that just goes into a []byte rather than
	// a full embed.FS
	_ "embed"

	"github.com/hashicorp/go-hclog"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

const (
	// RouterOSVersion contains the most recent qualified version
	// of the suite to be installed.
	RouterOSVersion = "7.17"

	// RouterPkgMIPSBE is the most recent qualified firmware package for
	// a root router to be installed to.  In general its safe to
	// use the most recent version, but this is what we tested.
	RouterPkgMIPSBE = "routeros-" + RouterOSVersion + "-mipsbe.npk"

	// RouterPkgARM is same as above, but for arm.
	RouterPkgARM = "routeros-" + RouterOSVersion + "-arm.npk"

	// RouterPkgARM64 is the same as above, but for arm64.
	RouterPkgARM64 = "routeros-" + RouterOSVersion + "-arm64.npk"

	// WifiPkgMIPSBE contains the wireless drivers qualified with the
	// matched version to the RouterPkg above.  These must
	// generally be updated in sync unless a specific assurance
	// has been obtained from Mikrotik that it is safe to split
	// the versions.
	WifiPkgMIPSBE = "wireless-" + RouterOSVersion + "-mipsbe.npk"

	// WifiPkgARM is same as above, but for arm.
	WifiPkgARM = "wireless-" + RouterOSVersion + "-arm.npk"

	// WifiPkgARM64 is the same as above, but for arm64".
	WifiPkgARM64 = "wireless-" + RouterOSVersion + "-arm64.npk"

	netinstallPkg  = "netinstall-" + RouterOSVersion + ".tar.gz"
	netinstallPath = "/usr/local/bin/netinstall-cli"

	provisionAddr = "192.168.88.2/24"
	targetAddr    = "192.168.88.1"

	// ImagePath is a location to store routerOS images into.
	ImagePath = "/usr/share/routeros"

	// BootstrapNetScoring is what the scoring box uses to bring
	// up the initial net which involves a special bootstrap VLAN.
	// This is necessary to bring up the network in a
	// non-conflicting way.
	BootstrapNetScoring = `/interface/vlan/add comment="Bootstrap Interface" interface=ether2 name=bootstrap0 vlan-id=2
/ip/address/add address=100.64.1.1/24 interface=bootstrap0`

	// BootstrapNetField contains the bootstrap of the network for
	// devices that are not the scoring box.
	BootstrapNetField = "/ip/dhcp-client/add interface=ether1 disabled=no add-default-route=no use-peer-dns=no use-peer-ntp=no"
)

// Installer wraps functionality associated with installation.
type Installer struct {
	l hclog.Logger

	pkgs         []string
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

// WithPackages configures what package should be installed
func WithPackages(p []string) InstallerOpt {
	return func(i *Installer) {
		i.pkgs = p
	}
}

// WithBootstrapNet configures the bootstrap configuration for the
// network device.
func WithBootstrapNet(s string) InstallerOpt {
	return func(i *Installer) {
		i.bootstrapCtx["network"] = s
	}
}

// WithFMS pulls in the relevant settings from the config that needs
// to be baked at netinstall time.
func WithFMS(c *config.FMSConfig) InstallerOpt {
	return func(i *Installer) {
		i.bootstrapCtx["AutoUser"] = c.AutoUser
		i.bootstrapCtx["AutoPass"] = c.AutoPass
		i.bootstrapCtx["ViewUser"] = c.ViewUser
		i.bootstrapCtx["ViewPass"] = c.ViewPass
		i.bootstrapCtx["AdminPass"] = c.AdminPass
	}
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

// TemplateConfig writes the configuration file to the specified
// location and does nothing else.
func (i *Installer) TemplateConfig(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.writeUserScript(f)
}

func (i *Installer) makeBootstrap() error {
	f, err := os.CreateTemp("", "*.rsc")
	if err != nil {
		return err
	}
	defer f.Close()
	i.bootstrap = f.Name()
	i.l.Info("Writing configuration to file", "path", i.bootstrap)
	return i.writeUserScript(f)
}

func (i *Installer) writeUserScript(w io.Writer) error {
	tpl, err := template.New("bootstrap.rsc").Parse(bootstrapCfg)
	if err != nil {
		return err
	}

	if err := tpl.Execute(w, i.bootstrapCtx); err != nil {
		return err
	}
	return nil
}

func (i *Installer) cleanup() error {
	return os.Remove(i.bootstrap)
}

func (i *Installer) doInstall() error {
	for p := range i.pkgs {
		i.pkgs[p] = filepath.Join(ImagePath, i.pkgs[p])
	}
	args := []string{
		"-s", i.bootstrap,
		"-r", "-a", targetAddr,
		"-o",
	}
	args = append(args, i.pkgs...)
	cmd := exec.Command(netinstallPath, args...)
	i.l.Debug("netinstall-cli", "command", cmd)

	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe

	cmd.Start()

	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "Successfully") {
				cmd.Process.Signal(syscall.SIGINT)
			}
			i.l.Info(scanner.Text())
		}
	}()
	return cmd.Wait()
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
