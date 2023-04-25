package shim

import (
	"errors"
)

// TLM is a Team Location Mapper that contains a static mapping.
type TLM struct {
	Mapping map[int]string
}

// GetFieldForTeam returns the current location for a given team number.
func (tlm *TLM) GetFieldForTeam(team int) (string, error) {
	mapping, ok := tlm.Mapping[team]
	if !ok {
		return "none:none", errors.New("no mapping for team")
	}
	return mapping, nil
}

// SetScheduleStep normally would advance the schedule when running a
// scheduled match.
func (tlm *TLM) SetScheduleStep(_ int) error { return nil }

// InsertOnDemandMap inserts an on-demand mapping that overrides any
// current schedule.  WARNING: This is immediate.
func (tlm *TLM) InsertOnDemandMap(m map[int]string) { tlm.Mapping = m }

// GetCurrentTeams returns the teams that are expected to be on the
// field at this time.
func (tlm *TLM) GetCurrentTeams() []int {
	ret := make([]int, len(tlm.Mapping))

	i := 0
	for team := range tlm.Mapping {
		ret[i] = team
	}

	return ret
}
