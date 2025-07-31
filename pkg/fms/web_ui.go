package fms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/flosch/pongo2/v6"
)

func (f *FMS) uiViewFieldHUD(w http.ResponseWriter, r *http.Request) {
	quadJSON, _ := json.Marshal(f.quads)
	f.doTemplate(w, r, "views/display/field-hud.p2", pongo2.Context{"quads": quadJSON})
}

func (f *FMS) uiViewLogin(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "login.p2", nil)
}

func (f *FMS) uiViewAdminLanding(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/admin/landing.p2", nil)
}

func (f *FMS) uiViewCurrentMap(w http.ResponseWriter, r *http.Request) {
	m, err := f.tlm.GetCurrentMapping()
	if err != nil {
		f.doTemplate(w, r, "errors/internal.p2", pongo2.Context{"error": err})
	}
	out, _ := json.Marshal(f.quads)
	ctx := pongo2.Context{
		"quads":    f.quads,
		"quadJSON": string(out),
		"active":   f.invertTLMMap(m),
		"teams":    f.c.Teams,
	}

	f.doTemplate(w, r, "views/map/current.p2", ctx)
}

func (f *FMS) uiViewStageMap(w http.ResponseWriter, r *http.Request) {
	stage, err := f.tlm.GetStageMapping()
	if err != nil {
		f.doTemplate(w, r, "errors/internal.p2", pongo2.Context{"error": err})
	}

	current, err := f.tlm.GetCurrentMapping()
	if err != nil {
		f.doTemplate(w, r, "errors/internal.p2", pongo2.Context{"error": err})
	}

	out, _ := json.Marshal(f.quads)
	ctx := pongo2.Context{
		"stage":    f.invertTLMMap(stage),
		"active":   f.invertTLMMap(current),
		"quads":    f.quads,
		"teams":    f.c.Teams,
		"roster":   f.c.SortedTeams(),
		"quadJSON": string(out),
	}

	f.doTemplate(w, r, "views/map/stage.p2", ctx)
}

func (f *FMS) uiViewUpdateStageMap(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		f.doTemplate(w, r, "errors/internal.p2", pongo2.Context{"error": err})
		return
	}

	m := make(map[int]string)
	for _, position := range f.quads {
		t := r.FormValue(position)
		if t == "0" {
			continue
		}
		tNum, _ := strconv.Atoi(t)
		m[tNum] = position
	}

	if err := f.tlm.InsertStageMapping(m); err != nil {
		f.l.Error("Error remapping teams!", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error inserting map: %s", err)
		return
	}
	http.Redirect(w, r, "/ui/admin/map/stage", http.StatusSeeOther)
}

func (f *FMS) uiViewCommitStageMap(w http.ResponseWriter, r *http.Request) {
	if err := f.tlm.CommitStagedMap(); err != nil {
		f.l.Error("Error commiting staged mapping!", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error commiting staged map: %s", err)
		return
	}
	f.tlm.InsertStageMapping(nil)
	http.Redirect(w, r, "/ui/admin/map/stage", http.StatusSeeOther)
}

func (f *FMS) uiViewOutOfBoxSetup(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/oob.p2", nil)
}

func (f *FMS) uiViewRosterForm(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/roster.p2", nil)
}

func (f *FMS) uiViewFieldForm(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/field.p2", pongo2.Context{"cfg": f.c})
}

func (f *FMS) uiViewNetWifi(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/net-wifi.p2", pongo2.Context{"cfg": f.c})
}

func (f *FMS) uiViewNetAdvanced(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/net-advanced.p2", pongo2.Context{"cfg": f.c})
}

func (f *FMS) uiViewIntegrations(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/integrations.p2", pongo2.Context{"cfg": f.c})
}

func (f *FMS) uiViewFlashDevice(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/flash-device.p2", nil)
}

func (f *FMS) uiViewBootstrapNet(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/setup/net-bootstrap.p2", nil)
}

func (f *FMS) uiViewNetReconcile(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/net/reconcile.p2", nil)
}
