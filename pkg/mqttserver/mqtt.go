package mqttserver

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

// StopHook is a function to be called when the MQTT server shuts
// down.
type StopHook func()

// Server binds the server's methods
type Server struct {
	l hclog.Logger
	s *mqtt.Server

	swg *sync.WaitGroup

	stopFeeds chan struct{}
	stopHooks []StopHook
}

// NewServer returns a running mqtt broker
func NewServer(opts ...Option) (*Server, error) {
	x := Server{
		l:         hclog.NewNullLogger(),
		s:         mqtt.New(&mqtt.Options{InlineClient: true}),
		stopFeeds: make(chan (struct{})),
	}

	for _, o := range opts {
		if err := o(&x); err != nil {
			return nil, err
		}
	}
	x.s.AddHook(newHook(x.l), nil)
	return &x, nil
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
	for _, hook := range s.stopHooks {
		hook()
	}
	return s.s.Close()
}

func (s *Server) metadataUpdater(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	parts := strings.Split(pk.TopicName, "/")
	if len(parts) != 3 {
		s.l.Warn("meta updater proc'd for non 3-part topic", "topic", pk.TopicName)
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	switch parts[2] {
	case "gizmo-meta":
		d := config.GizmoMeta{}
		if err := json.Unmarshal(pk.Payload, &d); err != nil {
			s.l.Warn("Error parsing gizmo-meta", "team", num, "error", err)
			return
		}
	}
}

// This returns the expected team number that should be communicating
// from this client based on the IP that they connected from.  Its not
// possible to identify the actual client with certainty from this
// point because the mqtt client ID is a client controlled value and
// as such cannot be trusted.
func teamNumberFromClient(cl *mqtt.Client) int {
	host, _, err := net.SplitHostPort(cl.Net.Remote)
	if err != nil {
		return -1
	}

	ip := net.ParseIP(host)
	expected := int(ip[13])*100 + int(ip[14])

	return expected
}
