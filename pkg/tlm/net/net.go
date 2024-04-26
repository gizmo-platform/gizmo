package net

import (
	"errors"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

// TLM is a Team Location Mapper that contains a static mapping.
type TLM struct {
	l hclog.Logger

	mapping    map[int]string
	mutex      sync.RWMutex
	metrics    *metrics.Metrics
	controller *config.Configurator
}

// New configures the TLM with the given options.
func New(opts ...Option) *TLM {
	t := new(TLM)
	t.mapping = make(map[int]string)

	for _, o := range opts {
		o(t)
	}

	return t
}

// GetFieldForTeam returns the current location for a given team number.
func (tlm *TLM) GetFieldForTeam(team int) (string, error) {
	tlm.mutex.RLock()
	mapping, ok := tlm.mapping[team]
	tlm.mutex.RUnlock()
	if !ok {
		return "none:none", errors.New("no mapping for team")
	}
	return mapping, nil
}

// InsertOnDemandMap inserts an on-demand mapping that overrides any
// current schedule.  WARNING: This is immediate.
func (tlm *TLM) InsertOnDemandMap(m map[int]string) error {
	tlm.mutex.Lock()
	defer tlm.mutex.Unlock()
	tlm.mapping = m
	tlm.metrics.ExportCurrentMatch(tlm.mapping)

	if err := tlm.controller.SyncTLM(tlm.mapping); err != nil {
		tlm.l.Error("Error syncronizing match state", "error", err)
		return err
	}

	// This is normally unsafe to do because it doesn't actually
	// check what state the fields are in before applying
	// configuration.  This is what the reconcile-net command is
	// provided for so that if you make a manual change to the
	// system its possible to reconcile that state back to what
	// the system expects is going on.  We do this because its
	// SIGNIFICANTLY faster than doing a state refresh and in the
	// general case, very few people even know how to take manual
	// control out from under the FMS in the first place.
	if err := tlm.controller.Converge(false, ""); err != nil {
		tlm.l.Error("Error converging fields", "error", err)
		return err
	}
	return nil
}

// GetCurrentMapping is a convenience function to retrieve the current
// mapping for the current match.
func (tlm *TLM) GetCurrentMapping() (map[int]string, error) { return tlm.mapping, nil }

// GetCurrentTeams returns the teams that are expected to be on the
// field at this time.
func (tlm *TLM) GetCurrentTeams() []int {
	tlm.metrics.ClearSchedule()
	ret := make([]int, len(tlm.mapping))

	i := 0
	tlm.mutex.RLock()
	for team := range tlm.mapping {
		ret[i] = team
	}
	tlm.mutex.RUnlock()

	return ret
}
