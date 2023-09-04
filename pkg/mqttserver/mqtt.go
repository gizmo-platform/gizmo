package mqttserver

import (
	"github.com/hashicorp/go-hclog"
	"github.com/mochi-co/mqtt/v2"
	"github.com/mochi-co/mqtt/v2/hooks/auth"
	"github.com/mochi-co/mqtt/v2/listeners"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// Server binds the server's methods
type Server struct {
	l hclog.Logger
	s *mqtt.Server

	stopFeeds chan struct{}
}

// NewServer returns a running mqtt broker
func NewServer(opts ...Option) (*Server, error) {
	x := new(Server)
	x.l = hclog.NewNullLogger()
	x.s = mqtt.New(nil)
	x.stopFeeds = make(chan (struct{}))

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	// Allow all, not necessarily a good idea but we control the
	// network and can assert we know who's on it.
	x.s.AddHook(new(auth.AllowHook), nil)
	return x, nil
}

// Serve binds and serves mqtt on the bound socket.  An error will be
// returned if the srever cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("MQTT is starting")
	if err := s.s.AddListener(listeners.NewTCP("default", bind, nil)); err != nil {
		return err
	}

	return s.s.Serve()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	s.l.Info("Stopping...")
	return s.s.Close()
}
