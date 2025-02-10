package fms

import (
	"context"
	"fmt"
	"io/fs"
	nhttp "net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/flosch/pongo2/v5"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/http"
)

// New configures and returns an FMS instance that is a runnable
// containing the TLM, webserver, and associated other components.
func New(opts ...Option) (*FMS, error) {
	sub, _ := fs.Sub(efs, "tpl")
	ldr := pongo2.NewFSLoader(sub)

	x := new(FMS)
	r := chi.NewRouter()
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
		FirmwareVersions: "0.1.6",
		Bootmodes:        "RAMDISK",
		DSVersions:       buildinfo.Version, // Always accept own version.
	}
	x.populateHUDVersions()

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	var err error
	x.s, err = http.NewServer(http.WithLogger(x.l), http.WithStartupWG(x.swg))
	if err != nil {
		x.l.Error("Could not create http server", "error", err)
		return nil, err
	}

	pongo2.RegisterFilter("valueok", x.filterValueOK)

	r.Handle("/static/*", nhttp.FileServer(nhttp.FS(sub)))
	r.Route("/gizmo/ds", func(r chi.Router) {
		r.Get("/{id}/config", x.gizmoConfig)
		r.Post("/{id}/meta", x.gizmoDSMetaReport)
	})
	r.Route("/gizmo/robot", func(r chi.Router) {
		r.Post("/{id}/meta", x.gizmoMetaReport)
	})
	r.Route("/admin", func(r chi.Router) {
		r.Get("/cfg/quads", x.configuredQuads)
		r.Post("/map/immediate", x.remapTeams)
		r.Post("/map/pcsm", x.remapTeamsPCSM)
		r.Get("/map/current", x.currentTeamMap)
		r.Route("/hud", func(hr chi.Router) {
			hr.Get("/", x.fieldHUD)
		})
	})
	r.Get("/metrics-sd", x.promSD)

	x.s.Mount("/", r)

	return x, nil
}

// Serve commences serving of the FMS endpoints.
func (f *FMS) Serve(bind string) error {
	go f.doConnectedUpkeep()
	go f.gizmoUDPServelet()
	f.swg.Done()

	return f.s.Serve(bind)
}

// Shutdown stops all components of the FMS.
func (f *FMS) Shutdown(ctx context.Context) error {
	f.stop <- struct{}{}
	return f.s.Shutdown(ctx)
}

func (f *FMS) templateErrorHandler(w nhttp.ResponseWriter, err error) {
	fmt.Fprintf(w, "Error while rendering template: %s\n", err)
}

func (f *FMS) doTemplate(w nhttp.ResponseWriter, r *nhttp.Request, tmpl string, ctx pongo2.Context) {
	if ctx == nil {
		ctx = pongo2.Context{}
	}
	t, err := f.tpl.FromCache(tmpl)
	if err != nil {
		f.templateErrorHandler(w, err)
		return
	}
	if err := t.ExecuteWriter(ctx, w); err != nil {
		f.templateErrorHandler(w, err)
	}
}

func (f *FMS) filterValueOK(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	list := strings.Split(param.String(), ",")
	for _, val := range list {
		if strings.TrimSpace(in.String()) == strings.TrimSpace(val) {
			return pongo2.AsValue(true), nil
		}
	}
	return pongo2.AsValue(false), nil
}

func (f *FMS) populateHUDVersions() {
	fw := os.Getenv("GIZMO_HUD_FWVERSIONS")
	if fw != "" {
		f.hudVersions.FirmwareVersions = fw
	}

	hw := os.Getenv("GIZMO_HUD_HWVERSIONS")
	if hw != "" {
		f.hudVersions.HardwareVersions = hw
	}

	bm := os.Getenv("GIZMO_HUD_BOOTMODES")
	if bm != "" {
		f.hudVersions.Bootmodes = bm
	}

	ds := os.Getenv("GIZMO_HUD_DSVERSIONS")
	if ds != "" {
		f.hudVersions.DSVersions = ds
	}
}
