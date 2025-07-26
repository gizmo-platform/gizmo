package netinstall

import (
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

// OptionSet is used to quickly refer to a set of multiple options
// that get returned from the options mapper.
type OptionSet uint8

const (
	// HardwareScoringBox refers to any hardware in the hEX line
	// which will be used as a router and network controller at
	// the scoring table.
	HardwareScoringBox OptionSet = iota + 1

	// HardwareScoringBoxLarge refers to the RB5009 when used as a
	// scoring router and network controller.
	HardwareScoringBoxLarge

	// HardwareFieldBox refers to a hAP AC device being used as a
	// field element.
	HardwareFieldBox

	// HardwareAuxiliary refers to a hAP lite device, which is not
	// capable of interacting with machines at distance, but is
	// still useful as a remote network element.
	HardwareAuxiliary
)

// Options returns the options that need to be passed to New() for a
// given device.
func (o OptionSet) Options() []InstallerOpt {
	opts := []InstallerOpt{}
	switch o {
	case HardwareScoringBox:
		opts = append(opts, WithBootstrapNet(BootstrapNetScoring))
		opts = append(opts, WithPackages([]string{
			RouterPkgMIPSBE,
			WifiPkgMIPSBE,
		}))
	case HardwareScoringBoxLarge:
		opts = append(opts, WithBootstrapNet(BootstrapNetScoring))
		opts = append(opts, WithPackages([]string{
			RouterPkgARM64,
			WifiPkgARM64,
		}))
	case HardwareFieldBox:
		opts = append(opts, WithBootstrapNet(BootstrapNetField))
		opts = append(opts, WithPackages([]string{
			RouterPkgARM,
			WifiPkgARM,
		}))
	case HardwareAuxiliary:
		opts = append(opts, WithBootstrapNet(BootstrapNetField))
		opts = append(opts, WithPackages([]string{
			RouterPkgMIPSBE,
			WifiPkgMIPSBE,
		}))
	}

	return opts
}

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

// WithEventStreamer provides an event streamer to the netinstaller so
// that log lines can be streamed to the frontend.
func WithEventStreamer(es EventStreamer) InstallerOpt {
	return func(i *Installer) {
		i.es = es
	}
}
