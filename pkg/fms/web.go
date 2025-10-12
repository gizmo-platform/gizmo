package fms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
	"github.com/gizmo-platform/gizmo/pkg/util"
)

func (f *FMS) apiGetConfig(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(f.c)
}

func (f *FMS) apiGetConfiguredQuads(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(f.quads)
}

func (f *FMS) apiGetTeamPresent(w http.ResponseWriter, r *http.Request) {
	field := chi.URLParam(r, "field")
	quad := chi.URLParam(r, "quad")
	f.dsPresentMutex.RLock()
	num := f.dsPresent["field"+field+":"+quad]
	f.dsPresentMutex.RUnlock()
	json.NewEncoder(w).Encode(num)
}

func (f *FMS) apiGetTeamPresentAll(w http.ResponseWriter, r *http.Request) {
	f.dsPresentMutex.RLock()
	defer f.dsPresentMutex.RUnlock()
	json.NewEncoder(w).Encode(f.dsPresent)
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

func (f *FMS) apiUpdateMapImmediate(w http.ResponseWriter, r *http.Request) {
	mapping := make(map[int]string)
	buf, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(buf, &mapping); err != nil {
		f.l.Warn("Error decoding on-demand mapping", "error", err, "body", string(buf))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Requests must be a map of team numbers for field locations")
		return
	}

	if err := f.tlm.InsertOnDemandMap(mapping); err != nil {
		f.l.Error("Error remapping teams!", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error inserting map: %s", err)
		return
	}
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
	if err := f.runSystemCommand(w, "sudo", "tzupdate"); err != nil {
		f.es.PublishError(err)
		return
	}
	if err := f.runSystemCommand(w, "sudo", "sv", "restart", "ntpd"); err != nil {
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Timezone Set")
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

func (f *FMS) apiUpdateCompatVer(w http.ResponseWriter, r *http.Request) {
	cTmp := new(config.FMSConfig)

	if err := json.NewDecoder(r.Body).Decode(&cTmp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// We do this rather than deserializing into the main config
	// struct to ensure that its not possible to rewrite other
	// unrelated parts of the config via this API.
	f.c.CompatHardwareVersions = cTmp.CompatHardwareVersions
	f.c.CompatFirmwareVersions = cTmp.CompatFirmwareVersions
	f.c.CompatDSBootmodes = cTmp.CompatDSBootmodes
	f.c.CompatDSVersions = cTmp.CompatDSVersions

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

func (f *FMS) apiDeviceFlashBegin(w http.ResponseWriter, r *http.Request) {
	optionSetID, _ := strconv.Atoi(r.URL.Query().Get("optionset"))

	opts := append(
		[]netinstall.InstallerOpt{},
		netinstall.WithLogger(f.l),
		netinstall.WithFMS(f.c),
		netinstall.WithEventStreamer(f.es),
	)
	opts = append(opts, netinstall.OptionSet(optionSetID).Options()...)
	f.netinst = netinstall.New(opts...)
	go func() {
		if err := f.netinst.Install(); err != nil && !strings.HasPrefix(err.Error(), "signal") {
			f.l.Warn("Error calling network installer", "error", err)
		}
	}()
}

func (f *FMS) apiDeviceFlashCancel(w http.ResponseWriter, r *http.Request) {
	f.netinst.Cancel()
	f.netinst = nil
}

func (f *FMS) apiInitNetController(w http.ResponseWriter, r *http.Request) {
	if err := f.net.SyncState(nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := f.net.Init(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *FMS) apiBootstrapBeginPhase0(w http.ResponseWriter, r *http.Request) {
	if err := f.net.BootstrapPhase0(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (f *FMS) apiBootstrapBeginPhase1(w http.ResponseWriter, r *http.Request) {
	if err := f.net.BootstrapPhase1(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (f *FMS) apiBootstrapBeginPhase2(w http.ResponseWriter, r *http.Request) {
	if err := f.net.BootstrapPhase2(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (f *FMS) apiBootstrapBeginPhase3(w http.ResponseWriter, r *http.Request) {
	if err := f.net.BootstrapPhase3(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (f *FMS) apiNetReconcile(w http.ResponseWriter, r *http.Request) {
	f.es.PublishActionStart("Network", "Reconciliation")
	if err := f.net.SyncState(nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	if err := f.net.Converge(false, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	if err := f.net.CycleRadio("2ghz"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}

	if err := f.net.CycleRadio("5ghz"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Network Reconciled")
}

func (f *FMS) apiZapController(w http.ResponseWriter, r *http.Request) {
	if err := f.net.Zap(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		f.es.PublishError(err)
		return
	}
	f.es.PublishActionComplete("Configuration Zapped")
}

func (f *FMS) apiFieldHUD(w http.ResponseWriter, r *http.Request) {
	type hudQuad struct {
		Color           string
		Actual          int
		Team            int
		GizmoConnected  bool
		GizmoFirmwareOK bool
		GizmoHardwareOK bool
		GizmoMeta       config.GizmoMeta
		DSConnected     bool
		DSBootOK        bool
		DSVersionOK     bool
		DSMeta          config.DSMeta
	}

	f.dsPresentMutex.RLock()
	defer f.dsPresentMutex.RUnlock()
	m, _ := f.tlm.GetCurrentMapping()
	tm := f.invertTLMMap(m)

	out := make([][]hudQuad, len(f.c.Fields))
	for _, field := range f.quads {
		parts := strings.Split(field, ":")
		n, err := strconv.Atoi(strings.TrimPrefix(parts[0], "field"))
		if err != nil {
			f.l.Error("Error decoding field number", "error", err)
			continue
		}
		n = n - 1
		team := tm[field]

		fTmp := hudQuad{
			Color:  parts[1],
			Team:   team,
			Actual: f.dsPresent[field],
		}
		f.connectedMutex.RLock()
		_, fTmp.GizmoConnected = f.connectedGizmo[team]
		_, fTmp.DSConnected = f.connectedDS[team]
		f.connectedMutex.RUnlock()

		f.metaMutex.RLock()
		fTmp.GizmoMeta = f.gizmoMeta[team]
		fTmp.GizmoHardwareOK = fTmp.GizmoMeta.HWVersionOK(f.c.CompatHardwareVersions)
		fTmp.GizmoFirmwareOK = fTmp.GizmoMeta.FWVersionOK(f.c.CompatFirmwareVersions)
		fTmp.DSMeta = f.dsMeta[team]
		fTmp.DSVersionOK = fTmp.DSMeta.VersionOK(f.c.CompatDSVersions)
		fTmp.DSBootOK = fTmp.DSMeta.BootmodeOK(f.c.CompatDSBootmodes)
		f.metaMutex.RUnlock()

		out[n] = append(out[n], fTmp)
	}
	json.NewEncoder(w).Encode(out)
}
