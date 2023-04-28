package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
)

// JSController defines the interface that the control server expects
// to be able to serve
type JSController interface {
	GetState(string) (*gamepad.Values, error)
	UpdateState(string) error
	FindController() (int, error)
	BindController(string, int) error
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
	reg *prometheus.Registry

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

	x.r.Handle("/metrics", promhttp.HandlerFor(x.reg, promhttp.HandlerOpts{Registry: x.reg}))

	x.r.Get("/robot/time", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, time.Now().Format(time.RFC3339))
		fmt.Fprint(w, "\r\n")
	})

	x.r.Get("/robot/data/gamepad/{team}", x.gamepadValueForTeam)
	x.r.Get("/robot/data/location/{team}", x.locationValueForTeam)
	x.r.Post("/robot/data/report/{team}", x.acceptDataForTeam)
	x.r.Post("/admin/map/immediate", x.remapTeams)
	x.r.Post("/admin/js/bind", x.bindJoystick)

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

func (s *Server) teamAndFIDFromRequest(r *http.Request) (int, string, error) {
	team, err := strconv.Atoi(chi.URLParam(r, "team"))
	if err != nil {
		return -1, "", err
	}

	fid, err := s.tlm.GetFieldForTeam(team)
	if err != nil {
		s.l.Warn("Team asked for field and they don't have one!", "team", team)
		return -1, "", err
	}
	return team, fid, nil
}
