package config

import (
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/fms"
)

// WithLogger sets the parent logging instance for the configurator
func WithLogger(l hclog.Logger) Option {
	return func(c *Configurator) { c.l = l.Named("configurator") }
}

// WithFMS provides the current FMS configuration to the system, which
// influences the components that are configured.
func WithFMS(fms *fms.Config) Option {
	return func(c *Configurator) { c.fc = fms }
}

// WithRouter sets the address on which the router can be contacted.
func WithRouter(ip string) Option {
	return func(c *Configurator) { c.routerAddr = ip }
}
