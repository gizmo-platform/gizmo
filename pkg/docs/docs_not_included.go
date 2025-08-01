//go:build !docs

package docs

import (
	"fmt"
	"net/http"
)

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Documentation is not available in this build!")
}

// MakeHandler just returns a handler that no matter what you put into
// it it tells you the docs aren't included in your current build.
func MakeHandler(_ string) http.Handler {
	h := new(handler)
	return h
}
