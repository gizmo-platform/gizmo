package mqttpusher

import (
	"github.com/hashicorp/go-hclog"
)

// Option enables variadic option passing to the server on startup.
type Option func(*Pusher) error

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(p *Pusher) error {
		p.l = l.Named("pusher")
		return nil
	}
}

// WithTeamLocationMapper sets the mapper instance for the server to
// get from team number and schedule step to the field that they're
// supposed to be on.
func WithTeamLocationMapper(t TeamLocationMapper) Option {
	return func(p *Pusher) error {
		p.tlm = t
		return nil
	}
}

// WithJSController sets the joystick controller for the
func WithJSController(jsc JSController) Option {
	return func(p *Pusher) error {
		p.jsc = jsc
		return nil
	}
}

// WithMQTTServer handles setting up the mqtt server address.
func WithMQTTServer(addr string) Option {
	return func(p *Pusher) error {
		p.addr = addr
		return nil
	}
}
