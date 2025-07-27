package fms

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

// Option enables variadic option passing to the server on startup.
type Option func(*FMS) error

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(f *FMS) error {
		f.l = l.Named("fms")
		return nil
	}
}

// WithTeamLocationMapper sets the mapper instance for the server to
// get from team number and schedule step to the field that they're
// supposed to be on.
func WithTeamLocationMapper(t TeamLocationMapper) Option {
	return func(f *FMS) error {
		f.tlm = t
		return nil
	}
}

// WithFMSConf generates all the quad data out of the config for the
// FMS itself.  It provides a more convenient system than using the
// direct Quad interface.
func WithFMSConf(c *config.FMSConfig) Option {
	return func(f *FMS) error {
		quads := []string{}
		for _, f := range c.Fields {
			for _, color := range []string{"red", "blue", "green", "yellow"} {
				quads = append(quads, fmt.Sprintf("field%d:%s", f.ID, color))
			}
		}
		f.quads = quads
		f.c = c
		return nil
	}
}

// WithStartupWG allows a waitgroup to be passed in so the server can
// notify when its finished with startup tasks to allow a nice message
// to be printed to the console.
func WithStartupWG(wg *sync.WaitGroup) Option {
	return func(f *FMS) error {
		wg.Add(1)
		f.swg = wg
		return nil
	}
}

// WithEventStreamer allows injecting a streaming backend for the FMS
// to serve over http.
func WithEventStreamer(es EventStreamer) Option {
	return func(f *FMS) error {
		f.es = es
		return nil
	}
}

// WithFileFetcher injects the backend that may be used to fetch
// restrited artifacts from Mikrotik.
func WithFileFetcher(fetcher FileFetcher) Option {
	return func(f *FMS) error {
		f.fetcher = fetcher
		return nil
	}
}

// WithNetController injects the backend network controller that will
// actually modify network state.
func WithNetController(nc NetController) Option {
	return func(f *FMS) error {
		f.net = nc
		return nil
	}
}
