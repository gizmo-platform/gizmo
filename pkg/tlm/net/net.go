package net

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

// state contains the state that the TLM may need to persist.
type state struct {
	Active map[int]string
	Stage  map[int]string
}

// TLM is a Team Location Mapper that contains a static mapping.
type TLM struct {
	l hclog.Logger

	state      state
	mutex      sync.RWMutex
	controller *config.Configurator

	savepath string

	swg  *sync.WaitGroup
	stop chan struct{}
}

// New configures the TLM with the given options.
func New(opts ...Option) *TLM {
	t := new(TLM)
	t.state.Active = make(map[int]string)
	t.state.Stage = make(map[int]string)
	t.stop = make(chan (struct{}))

	for _, o := range opts {
		o(t)
	}
	return t
}

// GetFieldForTeam returns the current location for a given team number.
func (tlm *TLM) GetFieldForTeam(team int) (string, error) {
	tlm.mutex.RLock()
	mapping, ok := tlm.state.Active[team]
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
	tlm.state.Active = m

	if err := tlm.controller.SyncTLM(tlm.state.Active); err != nil {
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

	if err := tlm.controller.CycleRadio("2ghz"); err != nil {
		tlm.l.Error("Error cycling radios", "error", err)
		return err
	}

	if err := tlm.SaveState(); err != nil {
		tlm.l.Warn("Error persisting match state", "error", err)
	}

	return nil
}

// GetCurrentMapping is a convenience function to retrieve the current
// mapping for the current match.
func (tlm *TLM) GetCurrentMapping() (map[int]string, error) { return tlm.state.Active, nil }

// InsertStageMapping is used to insert a mapping that is staged and
// can be committed at a later time.
func (tlm *TLM) InsertStageMapping(s map[int]string) error {
	tlm.mutex.Lock()
	defer tlm.mutex.Unlock()
	tlm.state.Stage = s
	if err := tlm.SaveState(); err != nil {
		tlm.l.Warn("Error persisting match state", "error", err)
	}

	return nil
}

// GetStageMapping allows peeking at the currently staged team
// mapping.
func (tlm *TLM) GetStageMapping() (map[int]string, error) { return tlm.state.Stage, nil }

// CommitStagedMap transfers staging to current, and applies it.  It
// does not clear the stage map!
func (tlm *TLM) CommitStagedMap() error {
	return tlm.InsertOnDemandMap(tlm.state.Stage)
}

// GetCurrentTeams returns the teams that are expected to be on the
// field at this time.
func (tlm *TLM) GetCurrentTeams() []int {
	ret := make([]int, len(tlm.state.Active))

	i := 0
	tlm.mutex.RLock()
	for team := range tlm.state.Active {
		ret[i] = team
	}
	tlm.mutex.RUnlock()

	return ret
}

// SaveState saves the TLM data to a file that can be recovered later.
func (tlm *TLM) SaveState() error {
	f, err := os.Create(tlm.savepath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tlm.state)
}

// RecoverState loads the TLM data from a file.
func (tlm *TLM) RecoverState() error {
	f, err := os.Open(tlm.savepath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&tlm.state)
}
