package mqttserver

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-co/mqtt/v2"
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

	swg *sync.WaitGroup

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
	x.s.AddHook(newHook(x.l), nil)
	return x, nil
}

// Serve binds and serves mqtt on the bound socket.  An error will be
// returned if the srever cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("MQTT is starting")
	if err := s.s.AddListener(listeners.NewTCP("default", bind, nil)); err != nil {
		return err
	}

	if s.swg != nil {
		s.swg.Done()
	}
	return s.s.Serve()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	s.l.Info("Stopping...")
	return s.s.Close()
}

// This returns 2 values, the actual team number that connected, and
// the number that we expected to show up based on the subnet that
// they connected from.
func teamNumberFromClient(cl *mqtt.Client) (int, int) {
	host, _, err := net.SplitHostPort(cl.Net.Remote)
	if err != nil {
		return -1, -1
	}

	ip := net.ParseIP(host)
	expected := int(ip[13])*100 + int(ip[14])

	name := cl.ID
	name = strings.TrimPrefix(name, "gizmo-ds")
	name = strings.TrimPrefix(name, "gizmo-")

	actual, err := strconv.Atoi(name)
	if err != nil {
		actual = -1
	}
	return expected, actual
}
