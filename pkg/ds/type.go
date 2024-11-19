package ds

import (
	"embed"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
	"github.com/gizmo-platform/gizmo/pkg/sysconf"
)

// DriverStation binds all methods related to the driver station task,
// which is a complicated component consisting of service supervision,
// configuration, and all the normal components that make up a field
// server.
type DriverStation struct {
	l hclog.Logger
	m *mqttserver.Server

	cfg  config.Config
	fCfg FieldConfig

	sc *sysconf.SysConf

	quit bool

	stop chan struct{}
}

// Option enables variadic configuration of the driver's station
// components.
type Option func(*DriverStation)

// A ConfigureStep performa various changes to the system to configure
// it.
type ConfigureStep func() error

//go:embed tpl/*.tpl
var efs embed.FS

// FieldConfig contains information obtained from the field and is
// used to determine whether or not the driver's station needs to
// adjust its mode of operation.
type FieldConfig struct {
	RadioMode    string
	RadioChannel string
	Field        int
	Location     string
}
