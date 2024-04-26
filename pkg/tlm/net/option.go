package net

import (
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

// Option configures the TLM
type Option func(t *TLM)

// WithLogger configures the logger for the team location mapper
func WithLogger(l hclog.Logger) Option {
	return func(t *TLM) {
		t.l = l.Named("tlm")
	}
}

// WithMetrics pushes a metrics instance to the TLM subsystem so that
// metrics can know what match is up.
func WithMetrics(m *metrics.Metrics) Option {
	return func(t *TLM) {
		t.metrics = m
	}
}

// WithController sets the network controller instance that will
// manage the network for this TLM.
func WithController(c *config.Configurator) Option {
	return func(t *TLM) {
		t.controller = c
	}
}
