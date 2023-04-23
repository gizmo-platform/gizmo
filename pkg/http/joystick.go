package http

import (
	"encoding/json"
	"net/http"
)

func (s *Server) bindJoystick(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)

	vals := &struct{ Field string }{}
	if err := dec.Decode(&vals); err != nil {
		s.l.Warn("Error binding field", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("could not parse field\n"))
		return
	}

	s.l.Debug("Attempting to bind joystick", "field", vals.Field)

	id, err := s.jsc.FindController()
	if err != nil {
		s.l.Warn("Error binding field", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		w.Write([]byte("\n"))
		return
	}

	s.l.Debug("Located controller for field", "field", vals.Field, "id", id)

	if err := s.jsc.BindController(vals.Field, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not bind controller\n"))
		w.Write([]byte(err.Error()))
		w.Write([]byte("\n"))
		return
	}

	w.Write([]byte("ok\n"))
}
