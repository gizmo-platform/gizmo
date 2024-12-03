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
	"time"

	"github.com/flosch/pongo2/v5"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/fms"
)

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetFieldForTeam(int) (string, error)
	GetCurrentMapping() (map[int]string, error)
	InsertOnDemandMap(map[int]string) error
}

type hudVersions struct {
	HardwareVersions string
	FirmwareVersions string
	Bootmodes        string
	DSVersions       string
}

// Server manages the HTTP serving components
type Server struct {
	r       chi.Router
	n       *http.Server
	l       hclog.Logger
	tlm     TeamLocationMapper
	swg     *sync.WaitGroup
	tpl     *pongo2.TemplateSet
	fmsConf fms.Config

	quads []string

	stop           chan struct{}
	connectedDS    map[int]time.Time
	connectedGizmo map[int]time.Time
	connectedMutex *sync.RWMutex
	gizmoMeta      map[int]config.GizmoMeta
	dsMeta         map[int]config.DSMeta
	metaMutex      *sync.RWMutex

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
	x.connectedDS = make(map[int]time.Time)
	x.connectedGizmo = make(map[int]time.Time)
	x.gizmoMeta = make(map[int]config.GizmoMeta)
	x.dsMeta = make(map[int]config.DSMeta)
	x.connectedMutex = new(sync.RWMutex)
	x.metaMutex = new(sync.RWMutex)
	x.stop = make(chan struct{})
	x.hudVersions = hudVersions{
		HardwareVersions: "GIZMO_V00_R6E,GIZMO_V1_0_R00",
		FirmwareVersions: "0.1.5",
		Bootmodes:        "RAMDISK",
		DSVersions:       buildinfo.Version, // Always accept own version.
	}
	x.populateHUDVersions()

	pongo2.RegisterFilter("valueok", x.filterValueOK)

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	x.r.Handle("/static/*", http.FileServer(http.FS(sub)))

	x.r.Route("/gizmo/ds", func(r chi.Router) {
		r.Get("/{id}/config", x.gizmoConfig)
		r.Post("/{id}/meta", x.gizmoDSMetaReport)
	})

	x.r.Route("/gizmo/robot", func(r chi.Router) {
		r.Post("/{id}/meta", x.gizmoMetaReport)
	})

	x.r.Route("/admin", func(r chi.Router) {
		r.Get("/cfg/quads", x.configuredQuads)
		r.Post("/map/immediate", x.remapTeams)
		r.Post("/map/pcsm", x.remapTeamsPCSM)
		r.Get("/map/current", x.currentTeamMap)
		r.Get("/hud", x.fieldHUD)
	})

	x.r.Get("/metrics-sd", x.promSD)

	return x, nil
}

// Serve binds and serves http on the bound socket.  An error will be
// returned if the server cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	go s.doConnectedUpkeep()
	s.n.Addr = bind
	s.n.Handler = s.r
	s.swg.Done()
	return s.n.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.l.Info("Stopping...")
	s.stop <- struct{}{}
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
