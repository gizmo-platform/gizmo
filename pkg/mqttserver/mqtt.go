package mqttserver

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// Server binds the server's methods
type Server struct {
	l hclog.Logger
	s *mqtt.Server

	swg *sync.WaitGroup

	stopFeeds chan struct{}
}

// ClientInfo contains information on clients that are connected, and
// if they're where they're supposed to be.
type ClientInfo struct {
	Number          int
	CorrectLocation bool
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
	l := listeners.NewTCP(listeners.Config{
		ID:      "tcp",
		Address: bind,
	})
	if err := s.s.AddListener(l); err != nil {
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

// Clients returns a list of all currently connected gizmo clients and
// whether or not they are where they're supposed to be.
func (s *Server) Clients() map[string]ClientInfo {
	out := make(map[string]ClientInfo)

	for id, cl := range s.s.Clients.GetAll() {
		if !strings.HasPrefix(id, "gizmo-") {
			continue
		}
		actualN, expectedN := teamNumberFromClient(cl)
		out[id] = ClientInfo{
			Number:          actualN,
			CorrectLocation: actualN == expectedN,
		}
	}
	return out
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
