package simple

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/bestrobotics/gizmo/pkg/metrics"
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

// WithMetrics pushes a metrics instance to the TLM subsystem so that
// metrics can know what match is up.
func WithMetrics(m *metrics.Metrics) Option {
	return func(t *TLM) {
		t.metrics = m
	}
}
