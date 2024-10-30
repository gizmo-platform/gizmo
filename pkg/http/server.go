package http

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/flosch/pongo2/v5"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gizmo-platform/gizmo/pkg/config"
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
	GizmoMeta(int) (bool, config.GizmoMeta)
	DSMeta(int) (bool, config.DSMeta)
}

type hudVersions struct {
	HardwareVersions string
	FirmwareVersions string
	Bootmodes        string
	DSVersions       string
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

	hudVersions hudVersions
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
	x.hudVersions = hudVersions{
		HardwareVersions: "GIZMO_V00_R6E,GIZMO_V1_0_R00",
		FirmwareVersions: "0.1.3",
		Bootmodes:        "RAMDISK",
		DSVersions:       "0.1.4",
	}
	x.populateHUDVersions()

	pongo2.RegisterFilter("valueok", x.filterValueOK)

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	x.r.Handle("/metrics", promhttp.HandlerFor(x.reg, promhttp.HandlerOpts{Registry: x.reg}))
	x.r.Handle("/static/*", http.FileServer(http.FS(sub)))

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

func (s *Server) filterValueOK(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	list := strings.Split(param.String(), ",")
	for _, val := range list {
		if strings.TrimSpace(in.String()) == strings.TrimSpace(val) {
			return pongo2.AsValue(true), nil
		}
	}
	return pongo2.AsValue(false), nil
}

func (s *Server) populateHUDVersions() {
	fw := os.Getenv("GIZMO_HUD_FWVERSIONS")
	if fw != "" {
		s.hudVersions.FirmwareVersions = fw
	}

	hw := os.Getenv("GIZMO_HUD_HWVERSIONS")
	if hw != "" {
		s.hudVersions.HardwareVersions = hw
	}

	bm := os.Getenv("GIZMO_HUD_BOOTMODES")
	if bm != "" {
		s.hudVersions.Bootmodes = bm
	}

	ds := os.Getenv("GIZMO_HUD_DSVERSIONS")
	if ds != "" {
		s.hudVersions.DSVersions = ds
	}
}
