package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) remapTeams(w http.ResponseWriter, r *http.Request) {
	mapping := make(map[int]string)

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&mapping); err != nil {
		s.l.Warn("Error decoding on-demand mapping", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Requests must be a map of team numbers for field locations")
		return
	}

	if err := s.tlm.InsertOnDemandMap(mapping); err != nil {
		s.l.Error("Error remapping teams!", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error inserting map: %s", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	s.l.Info("Immediately remapped teams!", "map", mapping)
}

func (s *Server) currentTeamMap(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	m, _ := s.tlm.GetCurrentMapping()
	enc.Encode(m)
}
