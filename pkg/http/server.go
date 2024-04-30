package http

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"sync"

	"github.com/flosch/pongo2/v5"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
)

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetFieldForTeam(int) (string, error)
	GetCurrentMapping() (map[int]string, error)
	InsertOnDemandMap(map[int]string) error
}

// MQTTServer contains the specific limited interface that needs to be
// made available for the HUD
type MQTTServer interface {
	Clients() map[string]mqttserver.ClientInfo
}

// Server manages the HTTP serving components
type Server struct {
	r   chi.Router
	n   *http.Server
	l   hclog.Logger
	tlm TeamLocationMapper
	mq  MQTTServer
	reg *prometheus.Registry
	swg *sync.WaitGroup
	tpl *pongo2.TemplateSet

	quads []string
}

//go:embed tpl
var efs embed.FS

// NewServer returns a running field controller.
func NewServer(opts ...Option) (*Server, error) {
	sub, _ := fs.Sub(efs, "tpl")
	ldr := pongo2.NewFSLoader(sub)

	x := new(Server)
	x.r = chi.NewRouter()
	x.n = &http.Server{}
	x.l = hclog.NewNullLogger()
	x.tpl = pongo2.NewSet("html", ldr)

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	x.r.Handle("/metrics", promhttp.HandlerFor(x.reg, promhttp.HandlerOpts{Registry: x.reg}))
	x.r.Handle("/static", http.FileServer(http.FS(sub)))

	x.r.Get("/admin/cfg/quads", x.configuredQuads)
	x.r.Post("/admin/map/immediate", x.remapTeams)
	x.r.Get("/admin/map/current", x.currentTeamMap)
	x.r.Get("/admin/hud", x.fieldHUD)

	return x, nil
}

// Serve binds and serves http on the bound socket.  An error will be
// returned if the server cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	s.n.Addr = bind
	s.n.Handler = s.r
	s.swg.Done()
	return s.n.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.l.Info("Stopping...")
	return s.n.Shutdown(ctx)
}

func (s *Server) templateErrorHandler(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "Error while rendering template: %s\n", err)
}

func (s *Server) doTemplate(w http.ResponseWriter, r *http.Request, tmpl string, ctx pongo2.Context) {
	if ctx == nil {
		ctx = pongo2.Context{}
	}
	t, err := s.tpl.FromCache(tmpl)
	if err != nil {
		s.templateErrorHandler(w, err)
		return
	}
	if err := t.ExecuteWriter(ctx, w); err != nil {
		s.templateErrorHandler(w, err)
	}
}
