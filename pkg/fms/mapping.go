package fms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (f *FMS) remapTeams(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusOK)
	f.l.Info("Immediately remapped teams!", "map", mapping)
}

func (f *FMS) currentTeamMap(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	m, _ := f.tlm.GetCurrentMapping()
	enc.Encode(m)
}

func (f *FMS) promSD(w http.ResponseWriter, r *http.Request) {
	type promTarget struct {
		Targets []string `json:"targets"`
	}

	m, _ := f.tlm.GetCurrentMapping()
	tgt := []string{}

	for id := range m {
		tgt = append(tgt, fmt.Sprintf("10.%d.%d.2:8080", int(id/100), id%100))
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode([]promTarget{{Targets: tgt}}); err != nil {
		f.l.Warn("Error writing prom sd", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
