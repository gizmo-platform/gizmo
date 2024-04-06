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

//go:embed mdbook/book/*
var efs embed.FS

// Handler returns the contents of the embedded docs filesystem.
func Handler() http.Handler {
	efs, _ := fs.Sub(efs, "mdbook/book")
	return http.FileServer(http.FS(efs))
}
