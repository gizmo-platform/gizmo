package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
)

// JSController defines the interface that the control server expects
// to be able to serve
type JSController interface {
	GetState(string) (*gamepad.Values, error)
	UpdateState(string) error
}

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetFieldForTeam(int) (string, error)
	SetScheduleStep(int) error
	InsertOnDemandMap(map[int]string)
}

// Server manages the HTTP serving components
type Server struct {
	r   chi.Router
	n   *http.Server
	l   hclog.Logger
	tlm TeamLocationMapper

	jsc JSController
}

// NewServer returns a running field controller.
func NewServer(opts ...Option) (*Server, error) {
	x := new(Server)
	x.r = chi.NewRouter()
	x.n = &http.Server{}
	x.l = hclog.NewNullLogger()

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	x.r.Get("/robot/time", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, time.Now().Format(time.RFC3339))
		fmt.Fprint(w, "\r\n")
	})

	x.r.Get("/robot/data/{team}", x.valueForTeam)

	x.r.Post("/admin/map/immediate", x.remapTeams)

	return x, nil
}

// Serve binds and serves http on the bound socket.  An error will be
// returned if the server cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	s.n.Addr = bind
	s.n.Handler = s.r
	return s.n.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.n.Shutdown(ctx)
}

func (s *Server) valueForTeam(w http.ResponseWriter, r *http.Request) {
	team, err := strconv.Atoi(chi.URLParam(r, "team"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fid, err := s.tlm.GetFieldForTeam(team)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.l.Warn("Team asked for field and they don't have one!", "team", team)
		return
	}

	vals, err := s.jsc.GetState(fid)
	if err != nil {
		s.l.Warn("Error retrieving controller state", "team", team, "fid", fid, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(vals)
}

func (s *Server) remapTeams(w http.ResponseWriter, r *http.Request) {
	mapping := make(map[int]string)

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&mapping); err != nil {
		s.l.Warn("Error decoding on-demand mapping", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Requests must be a map of team numbers for field locations")
		return
	}

	s.tlm.InsertOnDemandMap(mapping)
	w.WriteHeader(http.StatusOK)
	s.l.Info("Immediately remapped teams!", "map", mapping)
}
