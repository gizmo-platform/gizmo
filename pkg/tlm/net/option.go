package net

import (
	"sync"

	"github.com/hashicorp/go-hclog"

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

// WithStartupWG allows a waitgroup to be passed in so the server can
// notify when its finished startup tasks with a nice message on the
// console.
func WithStartupWG(w *sync.WaitGroup) Option {
	return func(t *TLM) {
		w.Add(1)
		t.swg = w
	}
}

// WithController sets the network controller instance that will
// manage the network for this TLM.
func WithController(c *config.Configurator) Option {
	return func(t *TLM) {
		t.controller = c
	}
}
