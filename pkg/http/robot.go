package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) gamepadValueForTeam(w http.ResponseWriter, r *http.Request) {
	team, fid, err := s.teamAndFIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	vals, err := s.jsc.GetState(fid)
	if err != nil {
		s.l.Warn("Error retrieving controller state", "team", team, "fid", fid, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(vals)
}

func (s *Server) locationValueForTeam(w http.ResponseWriter, r *http.Request) {
	_, fid, err := s.teamAndFIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	parts := strings.SplitN(fid, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
	enc := json.NewEncoder(w)
	enc.Encode(struct {
		Field    int
		Quadrant string
	}{
		Field:    fnum,
		Quadrant: strings.ToUpper(parts[1]),
	})
}

func (s *Server) acceptDataForTeam(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	team, fid, err := s.teamAndFIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	vals := struct {
		VBat              int
		RSSI              int
		WatchdogRemaining int
		WatchdogOK        bool
		PwrBoard          bool
		PwrPico           bool
		PwrGPIO           bool
		PwrMainA          bool
		PwrMainB          bool
	}{}

	if err := dec.Decode(&vals); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.l.Warn("Garbage stats from a robot", "error", err, "team", team, "field", fid)
		return
	}

	s.l.Debug("vals", "vals", vals)

	w.WriteHeader(http.StatusOK)
}
