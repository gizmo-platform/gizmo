package mqttserver

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/metrics"
)

// Option enables variadic option passing to the server on startup.
type Option func(*Server) error

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(s *Server) error {
		s.l = l.Named("mqtt")
		return nil
	}
}

// WithStartupWG allows a waitgroup to be passed in so the server can
// notify when its finished with startup tasks to allow a nice message
// to be printed to the console.
func WithStartupWG(wg *sync.WaitGroup) Option {
	return func(s *Server) error {
		wg.Add(1)
		s.swg = wg
		return nil
	}
}

// WithStats provides a registry for statistics to be retained and
// kept up with.
func WithStats(m *metrics.Metrics) Option {
	return func(s *Server) error {
		s.s.Subscribe("robot/+/stats", 0, m.MQTTCallback)

		s.stopHooks = append(s.stopHooks, func() {
			m.Shutdown()
		})
		return nil
	}
}
