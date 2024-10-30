package mqttserver

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

// Server binds the server's methods
type Server struct {
	l hclog.Logger
	s *mqtt.Server

	connectedGizmo      map[int]time.Time
	connectedGizmoMutex *sync.RWMutex
	connectedDS         map[int]time.Time
	connectedDSMutex    *sync.RWMutex

	gizmoMeta map[int]config.GizmoMeta
	dsMeta    map[int]config.DSMeta
	metaMutex *sync.RWMutex

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
	x := Server{
		l:                   hclog.NewNullLogger(),
		s:                   mqtt.New(&mqtt.Options{InlineClient: true}),
		stopFeeds:           make(chan (struct{})),
		connectedGizmo:      make(map[int]time.Time),
		connectedGizmoMutex: new(sync.RWMutex),
		connectedDS:         make(map[int]time.Time),
		connectedDSMutex:    new(sync.RWMutex),
		gizmoMeta:           make(map[int]config.GizmoMeta),
		dsMeta:              make(map[int]config.DSMeta),
		metaMutex:           new(sync.RWMutex),
	}

	for _, o := range opts {
		if err := o(&x); err != nil {
			return nil, err
		}
	}
	x.s.AddHook(newHook(x.l), nil)
	x.s.Subscribe("robot/+/gamepad", 0, x.lastSeenUpdater)
	x.s.Subscribe("robot/+/stats", 0, x.lastSeenUpdater)
	x.s.Subscribe("robot/+/ds-meta", 0, x.metadataUpdater)
	x.s.Subscribe("robot/+/gizmo-meta", 0, x.metadataUpdater)
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
	return s.s.Close()
}

// Clients returns a list of all currently connected gizmo clients and
// whether or not they are where they're supposed to be.
func (s *Server) Clients() map[string]ClientInfo {
	out := make(map[string]ClientInfo)

	s.connectedDSMutex.RLock()
	for id, t := range s.connectedDS {
		if time.Now().After(t.Add(time.Second * 3)) {
			continue
		}
		out[fmt.Sprintf("gizmo-ds%d", id)] = ClientInfo{
			Number: id,
		}
	}
	s.connectedDSMutex.RUnlock()
	s.connectedGizmoMutex.RLock()
	for id, t := range s.connectedGizmo {
		if time.Now().After(t.Add(time.Second * 3)) {
			continue
		}
		out[fmt.Sprintf("gizmo-%d", id)] = ClientInfo{
			Number: id,
		}
	}
	s.connectedGizmoMutex.RUnlock()
	return out
}

// GizmoMeta returns the most recent metadata received for a gizmo, or
// a nil struct and a bool that it hasn't been seen yet.
func (s *Server) GizmoMeta(team int) (bool, config.GizmoMeta) {
	s.metaMutex.RLock()
	defer s.metaMutex.RUnlock()
	meta, ok := s.gizmoMeta[team]
	return ok, meta
}

// DSMeta returns the most recent metadata received for a driver's
// station, or a nil struct and a bool that it hasn't been seen yet.
func (s *Server) DSMeta(team int) (bool, config.DSMeta) {
	s.metaMutex.RLock()
	defer s.metaMutex.RUnlock()
	meta, ok := s.dsMeta[team]
	return ok, meta
}

func (s *Server) lastSeenUpdater(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	parts := strings.Split(pk.TopicName, "/")
	if len(parts) != 3 {
		s.l.Warn("last seen proc'd for non 3-part topic", "topic", pk.TopicName)
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	switch parts[2] {
	case "gamepad":
		s.connectedDSMutex.Lock()
		s.connectedDS[num] = time.Now()
		s.connectedDSMutex.Unlock()
	case "stats":
		s.connectedGizmoMutex.Lock()
		s.connectedGizmo[num] = time.Now()
		s.connectedGizmoMutex.Unlock()
	}
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
	case "ds-meta":
		d := config.DSMeta{}
		if err := json.Unmarshal(pk.Payload, &d); err != nil {
			s.l.Warn("Error parsing ds-meta", "team", num, "error", err)
			return
		}
		s.metaMutex.Lock()
		s.dsMeta[num] = d
		s.metaMutex.Unlock()
	case "gizmo-meta":
		d := config.GizmoMeta{}
		if err := json.Unmarshal(pk.Payload, &d); err != nil {
			s.l.Warn("Error parsing gizmo-meta", "team", num, "error", err)
			return
		}
		s.metaMutex.Lock()
		s.gizmoMeta[num] = d
		s.metaMutex.Unlock()
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
