package metrics

import (
	"github.com/hashicorp/go-hclog"
)

// WithLogger provides a non-nil logger for the metrics instance to
// interact with.
func WithLogger(l hclog.Logger) Option {
	return func(m *Metrics) {
		m.l = l.Named("metrics")
	}
}

// WithBroker points the metrics exporter at the mqtt broker that the
// robots will be talking to.
func WithBroker(b string) Option {
	return func(m *Metrics) {
		m.broker = b
	}
}
