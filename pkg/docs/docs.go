//go:build docs

// Package docs contains an embedded filesystem containing the
// complete copy of the docs site that goes with a particulare
// release.  It does not include the server because it is expected to
// be served by a server from one or more other entrypoints.
package docs

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate mdbook build mdbook

//go:embed mdbook/book/html/*
var efs embed.FS

// MakeHandler returns the contents of the embedded docs filesystem.
func MakeHandler(path string) http.Handler {
	return func() http.Handler {
		efs, _ := fs.Sub(efs, "mdbook/book")
		return http.StripPrefix(path, http.FileServer(http.FS(efs)))
	}()
}
