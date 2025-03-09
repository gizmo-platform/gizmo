package fms

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/sysconf"
)

const (
	// We need to be before sysctls, which fire at 08.
	coresvc = "/etc/runit/core-services/06-gizmo.sh"

	promConf   = "/etc/prometheus/prometheus.yml"
	sysctlConf = "/etc/sysctl.conf"
	dhcpcdConf = "/etc/dhcpcd.conf"
	sddmConf   = "/etc/sddm.conf.d/gizmo.conf"

	// These are rpi5 specific and have to do with Void using
	// newer versions of most software than Raspbian, so these
	// will hopefully go away once Raspbian updates to a point
	// that they get broken as well.
	xorgConf         = "/etc/X11/xorg.conf.d/99-vc4.conf"
	brcmfmacModprobe = "/etc/modprobe.d/99-brcmfmac.conf"

	grafanaPromSrc   = "/usr/share/grafana/conf/provisioning/datasources/default.yaml"
	grafanaDashCfg   = "/usr/share/grafana/conf/provisioning/dashboards/default.yaml"
	grafanaDashGizmo = "/var/lib/grafana/dashboards/gizmo.json"
	grafanaDashHome  = "/usr/share/grafana/public/dashboards/home.json"
	grafanaDashLand  = "/var/lib/grafana/dashboards/home.json"

	pipeWireConfDir         = "/etc/pipewire/pipewire.conf.d/"
	pipeWirePulseConf       = "/usr/share/examples/pipewire/20-pipewire-pulse.conf"
	pipeWireWireplumberConf = "/usr/share/examples/wireplumber/10-wireplumber.conf"

	sudoersWheelInclude = "/etc/sudoers.d/wheel"

	adminIceWMTheme   = "/home/admin/.icewm/theme"
	adminIceWMStartup = "/home/admin/.icewm/startup"
)

//go:embed tpl/*
var efs embed.FS

// SetupTool does a lot of setup things for the FMS that are not
// something normal users would need to run.
type SetupTool struct {
	l hclog.Logger

	sc *sysconf.SysConf

	hwType []byte
}

// A ConfigureStep performa various changes to the system to configure
// it.
type ConfigureStep func() error

// NewSetupTool sets up a logger for the setup tool.
func NewSetupTool(l hclog.Logger) *SetupTool {
	return &SetupTool{
		l:      l.Named("setup-tool"),
		sc:     sysconf.New(sysconf.WithLogger(l.Named("setup-tool")), sysconf.WithFS(efs)),
		hwType: []byte("UNKNOWN"),
	}
}

// Install pulls in all the packages and things that we need the
// network for and is intended to be called during a CI build to
// retrieve everything that's required to construct the image.
// Install does not require the system services to be running.
func (st *SetupTool) Install() error {
	// This has to apply first because terraform is nonfree.
	if err := st.sc.InstallPkgs("void-repo-nonfree"); err != nil {
		return err
	}

	pkgs := []string{
		// System Components
		"chrony",
		"cloud-guest-utils",
		"docker",
		"dumb_runtime_dir",
		"iwd",
		"iwgtk",
		"tzupdate",

		// Multimedia
		"pipewire",
		"wireplumber",
		"pavucontrol",
		"volctl",

		// Graphical Session
		"firefox",
		"icewm",
		"mesa-dri",
		"sddm",
		"seatd",
		"xf86-video-fbdev",
		"xfce4-terminal",
		"xorg-fonts",
		"xorg-minimal",
		"xterm",
		"xsel",

		// Useful Tools
		"htop",
		"jq",
		"sv-helper",
		"tio",
		"tmux",

		// Direct Gizmo Dependencies
		"qemu-user-i386",
		"terraform",

		// Gizmo Telemetry
		"grafana",
		"prometheus",
	}
	return st.sc.InstallPkgs(pkgs...)
}

// SetupBoot installs the runtime hooks that startup the configuration
// jobs.
func (st *SetupTool) SetupBoot() error {
	return st.sc.Template(coresvc, "tpl/coresvc.sh.tpl", 0644, nil)
}

// Configure calls all the configure steps to configure the FMS workstation.
func (st *SetupTool) Configure() error {
	st.detectPlatform()

	steps := map[string]ConfigureStep{
		"sysctl":        st.configureSysctl,
		"hostname":      st.configureHostname,
		"dhcpcd":        st.configureDHCPCD,
		"pipewire":      st.configurePipewire,
		"sudo":          st.configureSudo,
		"sddm":          st.configureSDDM,
		"qemu":          st.configureQEMU,
		"icewm-session": st.configureIceWM,
		"prometheus":    st.configurePrometheus,
		"grafana":       st.configureGrafana,
		"services":      st.enableServices,
	}

	if bytes.HasPrefix(st.hwType, []byte("Raspberry Pi 5")) {
		steps["rpi5-xorg"] = st.rpi5configureXorg
		steps["rpi5-brcmfmac"] = st.rpi5configureBRCMFMAC
	}

	for id, step := range steps {
		st.l.Info("Configuring", "step", id)
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func (st *SetupTool) detectPlatform() {
	f, err := os.Open("/sys/firmware/devicetree/base/model")
	if err != nil {
		st.l.Warn("Could not check device tree for model")
		return
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		st.l.Warn("Could not read device tree for model")
	}

	st.hwType = b
}

func (st *SetupTool) configureSysctl() error {
	return st.sc.Template(sysctlConf, "tpl/sysctl.conf.tpl", 0644, nil)
}

func (st *SetupTool) configureHostname() error {
	return st.sc.SetHostname("gizmo-fms")
}

func (st *SetupTool) configureDHCPCD() error {
	return st.sc.Template(dhcpcdConf, "tpl/dhcpcd.conf.tpl", 0644, nil)
}

func (st *SetupTool) configurePipewire() error {
	if err := os.MkdirAll(pipeWireConfDir, 0755); err != nil {
		return err
	}

	if err := os.Link(pipeWireWireplumberConf, filepath.Join(pipeWireConfDir, filepath.Base(pipeWireWireplumberConf))); err != nil && !os.IsExist(err) {
		return err
	}

	if err := os.Link(pipeWirePulseConf, filepath.Join(pipeWireConfDir, filepath.Base(pipeWirePulseConf))); err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func (st *SetupTool) configureSDDM() error {
	return st.sc.Template(sddmConf, "tpl/sddm.conf.tpl", 0644, nil)
}

func (st *SetupTool) configureSudo() error {
	return st.sc.Template(sudoersWheelInclude, "tpl/sudoers.conf.tpl", 0644, nil)
}

func (st *SetupTool) configureIceWM() error {
	if err := st.sc.Template(adminIceWMTheme, "tpl/icewm_theme.tpl", 0644, nil); err != nil {
		return err
	}
	if err := st.sc.Template(adminIceWMStartup, "tpl/icewm_startup.tpl", 0755, nil); err != nil {
		return err
	}
	return nil
}

func (st *SetupTool) configureQEMU() error {
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
		// Extremely important, do not register the native ELF
		// handler to itself.  This is unrecoverable and you
		// have to reboot.
		return nil
	}

	name := "qemu-i386-static"
	magic := "\x7fELF\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x03\x00"
	mask := "\xff\xff\xff\xff\xff\xfe\xfe\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff"
	offset := "0"
	flags := "PFC"
	interpreter := "/usr/bin/qemu-i386-static"

	r := fmt.Sprintf(":%s:M:%s:%s:%s:%s:%s", name, offset, magic, mask, interpreter, flags)

	if err := exec.Command("/usr/bin/mount", "-t", "binfmt_misc", "binfmt_misc", "/proc/sys/fs/binfmt_misc").Run(); err != nil {
		st.l.Warn("Error mounting binfmt_misc", "error", err)
		return err
	}

	f, err := os.Create("/proc/sys/fs/binfmt_misc/register")
	if err != nil {
		st.l.Warn("Error opening /proc/sys/fs/binfmt_misc/register", "error", err)
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte(r)); err != nil {
		st.l.Warn("Error writing magic string", "error", err, "magic", r)
	}

	return nil
}

func (st *SetupTool) configurePrometheus() error {
	return st.sc.Template(promConf, "tpl/prometheus.yml.tpl", 0644, nil)
}

func (st *SetupTool) configureGrafana() error {
	if err := st.sc.Template(grafanaPromSrc, "tpl/grafana_default.yaml.tpl", 0644, nil); err != nil {
		return err
	}

	if err := st.sc.Template(grafanaDashCfg, "tpl/grafana_dashboards.yaml.tpl", 0644, nil); err != nil {
		return err
	}

	if err := st.sc.Template(grafanaDashGizmo, "tpl/grafana_dash_gizmo.json.tpl", 0644, nil); err != nil {
		return err
	}

	if err := st.sc.Template(grafanaDashHome, "tpl/grafana_dash_home.json.tpl", 0644, nil); err != nil {
		return err
	}

	if err := st.sc.Template(grafanaDashLand, "tpl/grafana_dash_home.json.tpl", 0644, nil); err != nil {
		return err
	}

	return nil
}

func (st *SetupTool) rpi5configureXorg() error {
	st.l.Warn("RPi5: Patching Graphics configuration")
	if err := st.sc.Template(xorgConf, "tpl/xorg.conf.tpl", 0644, nil); err != nil {
		return err
	}
	return nil
}

func (st *SetupTool) rpi5configureBRCMFMAC() error {
	// This is necessary because of this
	// https://github.com/raspberrypi/linux/issues/6049#issuecomment-2485431104.
	// It should be possible to remove this hack when either newer
	// firmware, newer kernel, or both are available.  In the
	// meantime though, the disabled feature is not significant.
	st.l.Warn("RPi5: Patching brcmfmac config")
	return st.sc.Template(brcmfmacModprobe, "tpl/brcmfmac.conf.tpl", 0644, nil)
}

func (st *SetupTool) enableServices() error {
	svcs := []string{
		"acpid",
		"dbus",
		"dhcpcd",
		"docker",
		"grafana",
		"iwd",
		"ntpd",
		"prometheus",
		"sddm",
		"seatd",
	}
	for _, s := range svcs {
		st.l.Info("Enabling Service", "service", s)
		st.sc.Enable(s)
	}
	return nil
}
