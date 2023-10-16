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
