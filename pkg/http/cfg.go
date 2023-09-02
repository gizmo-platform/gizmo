package http

import (
	"encoding/json"
	"net/http"
)

func (s *Server) configuredQuads(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.Encode(s.quads)
}
