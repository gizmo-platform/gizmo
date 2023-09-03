package mqttserver

import (
	"github.com/hashicorp/go-hclog"
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

// WithTeamLocationMapper sets the mapper instance for the server to
// get from team number and schedule step to the field that they're
// supposed to be on.
func WithTeamLocationMapper(t TeamLocationMapper) Option {
	return func(s *Server) error {
		s.tlm = t
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
