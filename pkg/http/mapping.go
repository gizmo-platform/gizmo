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

	s.tlm.InsertOnDemandMap(mapping)
	w.WriteHeader(http.StatusOK)
	s.l.Info("Immediately remapped teams!", "map", mapping)
}
