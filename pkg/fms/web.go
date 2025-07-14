package fms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/flosch/pongo2/v6"
)

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
	ctx := pongo2.Context{
		"quads":  f.quads,
		"active": f.invertTLMMap(m),
		"teams":  f.c.Teams,
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

	ctx := pongo2.Context{
		"stage":  f.invertTLMMap(stage),
		"active": f.invertTLMMap(current),
		"quads":  f.quads,
		"teams":  f.c.Teams,
		"roster": f.c.SortedTeams(),
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

func (f *FMS) apiGetCurrentMap(w http.ResponseWriter, r *http.Request) {
	m, _ := f.tlm.GetCurrentMapping()
	json.NewEncoder(w).Encode(m)
}

func (f *FMS) apiGetStageMap(w http.ResponseWriter, r *http.Request) {
	m, _ := f.tlm.GetStageMapping()
	json.NewEncoder(w).Encode(m)
}

func (f *FMS) apiUpdateStageMap(w http.ResponseWriter, r *http.Request) {
	mapping := make(map[int]string)
	buf, _ := io.ReadAll(r.Body)

	if err := json.Unmarshal(buf, &mapping); err != nil {
		f.l.Warn("Error decoding on-demand mapping", "error", err, "body", string(buf))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Requests must be a map of team numbers for field locations")
		return
	}

	if err := f.tlm.InsertStageMapping(mapping); err != nil {
		f.l.Error("Error remapping teams!", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error inserting map: %s", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (f *FMS) apiCommitStageMap(w http.ResponseWriter, r *http.Request) {
	if err := f.tlm.CommitStagedMap(); err != nil {
		f.l.Error("Error commiting staged mapping!", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error commiting staged map: %s", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (f *FMS) invertTLMMap(m map[int]string) map[string]int {
	out := make(map[string]int)
	for k, v := range m {
		out[v] = k
	}
	return out
}
