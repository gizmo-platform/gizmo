package fms

import (
	"encoding/json"
	"net/http"
)

func (f *FMS) configuredQuads(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(f.quads)
}
