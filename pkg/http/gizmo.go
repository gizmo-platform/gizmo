package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/ds"
)

func (s *Server) doConnectedUpkeep() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-s.stop:
			ticker.Stop()
			return
		case <-ticker.C:
			s.connectedMutex.Lock()
			s.metaMutex.Lock()
			for id, expiry := range s.connectedDS {
				if time.Now().After(expiry) {

					delete(s.connectedDS, id)
					delete(s.dsMeta, id)
				}
			}
			for id, expiry := range s.connectedGizmo {
				if time.Now().After(expiry) {
					delete(s.connectedGizmo, id)
					delete(s.gizmoMeta, id)
				}
			}
			s.metaMutex.Unlock()
			s.connectedMutex.Unlock()
		}
	}
}

func (s *Server) gizmoConfig(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	location, err := s.tlm.GetFieldForTeam(team)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	parts := strings.SplitN(location, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))

	d := ds.FieldConfig{
		RadioMode:    s.fmsConf.RadioMode,
		RadioChannel: s.fmsConf.Fields[fnum-1].Channel,
		Field:        fnum,
		Location:     strings.ToUpper(parts[1]),
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) gizmoDSMetaReport(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		s.l.Warn("Bad DS Meta report", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	d := config.DSMeta{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		s.l.Warn("Error deserializing DS Meta report", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.connectedMutex.Lock()
	s.connectedDS[team] = time.Now().Add(time.Second * 5)
	s.connectedMutex.Unlock()

	s.metaMutex.Lock()
	s.dsMeta[team] = d
	s.metaMutex.Unlock()
}

func (s *Server) gizmoMetaReport(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.l.Warn("Bad Gizmo Meta Report", "error", err)
		return
	}

	d := config.GizmoMeta{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.l.Warn("Error deserializing Gizmo Meta report", "error", err)
		return
	}

	s.connectedMutex.Lock()
	s.connectedGizmo[team] = time.Now().Add(time.Second * 5)
	s.connectedMutex.Unlock()

	s.metaMutex.Lock()
	s.gizmoMeta[team] = d
	s.metaMutex.Unlock()
}
