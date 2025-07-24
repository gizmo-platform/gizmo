package fms

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/util"
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

func (f *FMS) apiGetConfig(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(f.c)
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

func (f *FMS) apiFetchTools(w http.ResponseWriter, r *http.Request) {
	if err := f.fetcher.FetchTools(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (f *FMS) apiFetchPackages(w http.ResponseWriter, r *http.Request) {
	if err := f.fetcher.FetchPackages(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (f *FMS) apiSetTimezone(w http.ResponseWriter, r *http.Request) {
	// This is a bad antipattern, but you need to be root to
	// modify the clock and to adjust the timezone links, and this
	// is the better approach than running the entire Gizmo
	// process with elevated permissions.  We could use a setuid
	// helper binary here, but that would be clunky and usually
	// leads to more security problems than it solves.
	f.es.PublishActionStart("Set Timezone", "tzupdate")
	f.runSystemCommand(w, "sudo", "tzupdate")
}

func (f *FMS) apiUpdateRoster(w http.ResponseWriter, r *http.Request) {
	teams := make(map[int]*config.Team)

	if err := json.NewDecoder(r.Body).Decode(&teams); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.es.PublishActionStart("Roster Change", "from web")

	vlan := 500
	for num, t := range teams {
		t.VLAN = vlan
		t.SSID = strings.ReplaceAll(uuid.New().String(), "-", "")
		t.PSK = strings.ReplaceAll(uuid.New().String(), "-", "")
		t.CIDR = fmt.Sprintf("10.%d.%d.0/24", int(num/100), num%100)
		t.GizmoMAC = util.NumberToMAC(num, 0).String()
		t.DSMAC = util.NumberToMAC(num, 1).String()

		vlan++
	}

	f.c.ReconcileTeams(teams)
	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Roster Change")
}

func (f *FMS) apiUpdateNetWifi(w http.ResponseWriter, r *http.Request) {
	cTmp := new(config.FMSConfig)

	if err := json.NewDecoder(r.Body).Decode(&cTmp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// We do this rather than deserializing into the main config
	// struct to ensure that its not possible to rewrite other
	// unrelated parts of the config via this API.
	f.c.InfrastructureVisible = cTmp.InfrastructureVisible
	f.c.InfrastructureSSID = cTmp.InfrastructureSSID
	f.c.InfrastructurePSK = cTmp.InfrastructurePSK

	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) apiUpdateAdvancedNet(w http.ResponseWriter, r *http.Request) {
	cTmp := new(config.FMSConfig)

	if err := json.NewDecoder(r.Body).Decode(&cTmp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// We do this rather than deserializing into the main config
	// struct to ensure that its not possible to rewrite other
	// unrelated parts of the config via this API.
	f.c.FixedDNS = cTmp.FixedDNS
	f.c.AdvancedBGPAS = cTmp.AdvancedBGPAS
	f.c.AdvancedBGPIP = cTmp.AdvancedBGPIP
	f.c.AdvancedBGPPeerIP = cTmp.AdvancedBGPPeerIP
	f.c.AdvancedBGPVLAN = cTmp.AdvancedBGPVLAN

	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) apiUpdateIntegrations(w http.ResponseWriter, r *http.Request) {
	integrations := config.IntegrationSlice{}

	if err := json.NewDecoder(r.Body).Decode(&integrations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	f.c.Integrations = integrations
	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) apiFieldAdd(w http.ResponseWriter, r *http.Request) {
	field := new(config.Field)

	if err := json.NewDecoder(r.Body).Decode(field); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	if _, exists := f.c.Fields[field.ID-1]; exists {
		http.Error(w, "Already Exists!", http.StatusConflict)
		return
	}

	f.c.Fields[field.ID-1] = field
	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) apiFieldUpdate(w http.ResponseWriter, r *http.Request) {
	field := new(config.Field)
	fNum, _ := strconv.Atoi(chi.URLParam(r, "id"))

	if err := json.NewDecoder(r.Body).Decode(field); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	if _, exists := f.c.Fields[fNum]; !exists {
		http.Error(w, "Does not exist!", http.StatusConflict)
		return
	}

	f.c.Fields[field.ID] = field
	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) apiFieldDelete(w http.ResponseWriter, r *http.Request) {
	fNum, _ := strconv.Atoi(chi.URLParam(r, "id"))
	delete(f.c.Fields, fNum-1)
	if err := f.c.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	f.es.PublishActionComplete("Configuration Save")
}

func (f *FMS) runSystemCommand(w http.ResponseWriter, exe string, args ...string) error {
	flusher, flushAvailable := w.(http.Flusher)
	cmd := exec.Command(exe, args...)
	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe
	cmd.Start()

	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			w.Write(scanner.Bytes())
			w.Write([]byte("\r\n"))
			if flushAvailable {
				flusher.Flush()
			}
		}
	}()
	err := cmd.Wait()
	w.Write([]byte("\r\n"))
	return err
}

func (f *FMS) invertTLMMap(m map[int]string) map[string]int {
	out := make(map[string]int)
	for k, v := range m {
		out[v] = k
	}
	return out
}
