package shim

import (
	"github.com/hashicorp/go-hclog"
)

// Option configures the TLM
type Option func(t *TLM)

// WithLogger configures the logger for the team location mapper
func WithLogger(l hclog.Logger) Option {
	return func(t *TLM) {
		t.l = l.Named("tlm")
	}
}
