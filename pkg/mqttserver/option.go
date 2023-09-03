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
