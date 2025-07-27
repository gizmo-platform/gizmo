package fms

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	nhttp "net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/http"
)

//go:embed ui/*
var uifs embed.FS

// New configures and returns an FMS instance that is a runnable
// containing the TLM, webserver, and associated other components.
func New(opts ...Option) (*FMS, error) {
	sub, _ := fs.Sub(uifs, "ui/p2")
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
	pongo2.RegisterFilter("split", x.filterSplit)
	pongo2.RegisterFilter("teamName", x.filterTeamName)

	sfs, _ := fs.Sub(uifs, "ui")
	r.Handle("/static/*", nhttp.FileServer(nhttp.FS(sfs)))
	r.Get("/login", x.uiViewLogin)
	r.Route("/gizmo/ds", func(r chi.Router) {
		r.Get("/{id}/config", x.gizmoConfig)
		r.Post("/{id}/meta", x.gizmoDSMetaReport)
	})
	r.Route("/gizmo/robot", func(r chi.Router) {
		r.Post("/{id}/meta", x.gizmoMetaReport)
	})
	r.Route("/admin", func(r chi.Router) {
		r.Get("/", x.uiViewAdminLanding)
		r.Get("/cfg/quads", x.configuredQuads)
		r.Post("/map/immediate", x.remapTeams)
		r.Post("/map/pcsm", x.remapTeamsPCSM)
		r.Get("/map/current", x.currentTeamMap)
		r.Route("/hud", func(hr chi.Router) {
			hr.Get("/", x.fieldHUD)
		})
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/config", x.apiGetConfig)
		r.Get("/eventstream", x.es.Handler)
		r.Route("/field", func(r chi.Router) {
			r.Get("/configured-quads", x.configuredQuads)
		})
		r.Route("/map", func(r chi.Router) {
			r.Get("/current", x.apiGetCurrentMap)
			r.Get("/stage", x.apiGetStageMap)
			r.Post("/stage", x.apiUpdateStageMap)
			r.Post("/commit-stage", x.apiCommitStageMap)
		})

		r.Route("/setup", func(r chi.Router) {
			r.Post("/fetch-tools", x.apiFetchTools)
			r.Post("/fetch-packages", x.apiFetchPackages)
			r.Post("/set-timezone", x.apiSetTimezone)
			r.Post("/update-roster", x.apiUpdateRoster)
			r.Post("/update-wifi", x.apiUpdateNetWifi)
			r.Post("/update-advanced-net", x.apiUpdateAdvancedNet)
			r.Post("/update-integrations", x.apiUpdateIntegrations)

			r.Route("/field", func(r chi.Router) {
				r.Post("/", x.apiFieldAdd)
				r.Put("/{id}", x.apiFieldUpdate)
				r.Delete("/{id}", x.apiFieldDelete)
			})

			r.Route("/device", func(r chi.Router) {
				r.Post("/begin-flash", x.apiDeviceFlashBegin)
				r.Post("/cancel-flash", x.apiDeviceFlashCancel)
			})

			r.Route("/net", func(r chi.Router) {
				r.Post("/init", x.apiInitNetController)
				r.Route("/bootstrap", func(r chi.Router) {
					r.Post("/phase0", x.apiBootstrapBeginPhase0)
					r.Post("/phase1", x.apiBootstrapBeginPhase1)
					r.Post("/phase2", x.apiBootstrapBeginPhase2)
					r.Post("/phase3", x.apiBootstrapBeginPhase3)
				})
			})
		})

		r.Route("/net", func(r chi.Router) {
			r.Post("/reconcile", x.apiNetReconcile)
		})
	})

	r.Route("/ui", func(r chi.Router) {
		r.Route("/admin", func(r chi.Router) {
			r.Get("/", x.uiViewAdminLanding)

			r.Route("/map", func(r chi.Router) {
				r.Get("/current", x.uiViewCurrentMap)
				r.Get("/stage", x.uiViewStageMap)
				r.Post("/stage", x.uiViewUpdateStageMap)
				r.Post("/commit-stage", x.uiViewCommitStageMap)
			})

			r.Route("/setup", func(r chi.Router) {
				r.Get("/oob", x.uiViewOutOfBoxSetup)
				r.Get("/roster", x.uiViewRosterForm)
				r.Get("/field", x.uiViewFieldForm)
				r.Get("/net-wifi", x.uiViewNetWifi)
				r.Get("/net-advanced", x.uiViewNetAdvanced)
				r.Get("/integrations", x.uiViewIntegrations)
				r.Get("/flash-device", x.uiViewFlashDevice)
				r.Get("/bootstrap-net", x.uiViewBootstrapNet)
			})

			r.Route("/net", func(r chi.Router) {
				r.Get("/reconcile", x.uiViewNetReconcile)
			})
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
		ctx = pongo2.Context{"shownav": true}
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

func (f *FMS) filterValueOK(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	list := strings.Split(param.String(), ",")
	for _, val := range list {
		if strings.TrimSpace(in.String()) == strings.TrimSpace(val) {
			return pongo2.AsValue(true), nil
		}
	}
	return pongo2.AsValue(false), nil
}

func (f *FMS) filterSplit(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	list := strings.Split(in.String(), param.String())
	return pongo2.AsValue(list), nil
}

func (f *FMS) filterTeamName(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	t, ok := in.Interface().(*config.Team)
	if !ok {
		f.l.Error("Something that wasn't a team got passed to the teamName filter", "in", in.Interface())
		return pongo2.AsValue(""), &pongo2.Error{Sender: "filter:teamName"}
	}
	return pongo2.AsValue(t.Name), nil
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
