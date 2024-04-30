package http

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gizmo-platform/gizmo/pkg/fms"
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

// WithFMSConf generates all the quad data out of the config for the
// FMS itself.  It provides a more convenient system than using the
// direct Quad interface.
func WithFMSConf(c fms.Config) Option {
	return func(s *Server) error {
		quads := []string{}
		for _, f := range c.Fields {
			for _, color := range []string{"red", "blue", "green", "yellow"} {
				quads = append(quads, fmt.Sprintf("field%d:%s", f.ID, color))
			}
		}
		s.quads = quads
		return nil
	}
}

// WithQuads tells the server what quadrants are available to
// configure.
func WithQuads(q []string) Option {
	return func(s *Server) error {
		s.quads = q
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

// WithMQTTServer provides the specific MQTTServer that's fronted by
// this webserver.
func WithMQTTServer(mq MQTTServer) Option {
	return func(s *Server) error {
		s.mq = mq
		return nil
	}
}
