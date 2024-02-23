package mqttpusher

import (
	"sync"

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

// WithMQTTServer handles setting up the mqtt server address.
func WithMQTTServer(addr string) Option {
	return func(p *Pusher) error {
		p.addr = addr
		return nil
	}
}

// WithStartupWG allows a waitgroup to be passed in so the server can
// notify when its finished startup tasks with a nice message on the
// console.
func WithStartupWG(w *sync.WaitGroup) Option {
	return func(p *Pusher) error {
		w.Add(1)
		p.swg = w
		return nil
	}
}

// WithQuadMap provides the mapping of each field ID/quad that this
// pusher is responsible for to the local joystick ID for that quad.
func WithQuadMap(q map[string]int) Option {
	return func(p *Pusher) error {
		p.quadMap = q
		return nil
	}
}
