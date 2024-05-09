package http

import (
	"encoding/json"
	"net/http"
)

func (s *Server) configuredQuads(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.quads)
}
