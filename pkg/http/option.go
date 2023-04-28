package http

import (
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
)

// Option enables variadic option passing to the server on startup.
type Option func(*Server) error

// WithPrometheusRegistry sets the Prometheus registry for the server
func WithPrometheusRegistry(reg *prometheus.Registry) Option {
	return func(s *Server) error {
		s.reg = reg
		return nil
	}
}

// WithJSController sets the joystick controller for the
func WithJSController(jsc JSController) Option {
	return func(s *Server) error {
		s.jsc = jsc
		return nil
	}
}

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(s *Server) error {
		s.l = l.Named("web")
		return nil
	}
}

// WithTeamLocationMapper sets the mapper instance for the server to
// get from team number and schedule step to the field that they're
// supposed to be on.
func WithTeamLocationMapper(t TeamLocationMapper) Option {
	return func(s *Server) error {
		s.tlm = t
		return nil
	}
}
