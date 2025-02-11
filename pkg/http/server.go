package http

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
)

// Server manages the HTTP serving components
type Server struct {
	r   chi.Router
	n   *http.Server
	l   hclog.Logger
	swg *sync.WaitGroup
}

// NewServer returns a running field controller.
func NewServer(opts ...Option) (*Server, error) {
	x := new(Server)
	x.r = chi.NewRouter()
	x.n = &http.Server{}
	x.l = hclog.NewNullLogger()

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	return x, nil
}

// Serve binds and serves http on the bound socket.  An error will be
// returned if the server cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	s.n.Addr = bind
	s.n.Handler = s.r
	s.swg.Done()
	return s.n.ListenAndServe()
}

// Mount attaches a set of routes to the subpath specified by the path
// argument.
func (s *Server) Mount(path string, router chi.Router) {
	s.r.Mount(path, router)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.l.Info("Stopping...")
	return s.n.Shutdown(ctx)
}
