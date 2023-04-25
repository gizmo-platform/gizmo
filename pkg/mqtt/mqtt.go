package mqtt

import (
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-co/mqtt/v2"
	"github.com/mochi-co/mqtt/v2/hooks/auth"
	"github.com/mochi-co/mqtt/v2/listeners"
	"github.com/rs/zerolog"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// JSController defines the interface that the control server expects
// to be able to serve
type JSController interface {
	GetState(string) (*gamepad.Values, error)
}

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetCurrentTeams() []int
	GetFieldForTeam(int) (string, error)
}

// Server binds the server's methods
type Server struct {
	l hclog.Logger
	s *mqtt.Server

	tlm TeamLocationMapper
	jsc JSController

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
	if err := s.s.AddListener(listeners.NewTCP("default", bind, nil)); err != nil {
		return err
	}

	return s.s.Serve()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	return s.s.Close()
}

func (s *Server) publishGamepadForTeam(team int) {
	fid, err := s.tlm.GetFieldForTeam(team)
	if err != nil {
		s.l.Warn("Trying to send gamepad state for an unmapped team!", "team", team)
		return
	}

	vals, err := s.jsc.GetState(fid)
	if err != nil {
		s.l.Warn("Error retrieving controller state", "team", team, "fid", fid, "error", err)
		return
	}

	bytes, err := json.Marshal(vals)
	if err != nil {
		s.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
		return
	}

	topic := path.Join("robot", strconv.Itoa(team), "gamepad")
	if err := s.s.Publish(topic, bytes, false, 0); err != nil {
		s.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
	}
}

func (s *Server) publishLocationForTeam(team int) {
	fid, err := s.tlm.GetFieldForTeam(team)
	if err != nil {
		s.l.Warn("Trying to send gamepad state for an unmapped team!", "team", team)
		return
	}

	parts := strings.SplitN(fid, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
	vals := struct {
		Field    int
		Quadrant string
	}{
		Field:    fnum,
		Quadrant: strings.ToUpper(parts[1]),
	}

	bytes, err := json.Marshal(vals)
	if err != nil {
		s.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
		return
	}

	topic := path.Join("robot", strconv.Itoa(team), "location")
	if err := s.s.Publish(topic, bytes, false, 0); err != nil {
		s.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
	}
}

// StartControlPusher spins off workers that push the data to machines
// as specified by the team location mapper.
func (s *Server) StartControlPusher() {
	ctrlTicker := time.NewTicker(time.Millisecond * 40)
	locTicker := time.NewTicker(time.Second * 5)

	go func() {
		for {
			select {
			case <-s.stopFeeds:
				ctrlTicker.Stop()
				locTicker.Stop()
				return
			case <-locTicker.C:
				for _, t := range s.tlm.GetCurrentTeams() {
					s.publishLocationForTeam(t)
				}
			case <-ctrlTicker.C:
				for _, t := range s.tlm.GetCurrentTeams() {
					s.publishGamepadForTeam(t)
				}
			}
		}
	}()
}

// StopControlPusher closes down the workers that publish information
// into the mqtt streams.
func (s *Server) StopControlPusher() {
	s.stopFeeds <- struct{}{}
}
