package fms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// This file includes integration code to make the Gizmo work well
// with the BEST Robotics PC Scoring Manager (PCSM).  PCSM serializes
// its internal match format and sends that across the wire, which
// then needs to be deserialized and converted into a TLM mapping.

type pcsmMatch struct {
	Number int `json:"matchNumber"`
	Fields []pcsmField
}

type pcsmField struct {
	Number int `json:"fieldNumber"`
	Teams  []pcsmTeam
}

type pcsmTeam struct {
	Number   int `json:"teamNumber"`
	Name     string
	Quadrant string
}

func (p *pcsmMatch) toTLM() map[int]string {
	out := make(map[int]string, len(p.Fields)*4)

	for _, field := range p.Fields {
		for _, t := range field.Teams {
			if t.Number == 0 {
				continue
			}
			out[t.Number] = fmt.Sprintf("field%d:%s", field.Number, strings.ToLower(t.Quadrant))
		}
	}

	return out
}

func (f *FMS) remapTeamsPCSM(w http.ResponseWriter, r *http.Request) {
	if !f.c.Integrations.Enabled(IntegrationPCSM) {
		w.WriteHeader(http.StatusPreconditionFailed)
		w.Write([]byte("Integration is not enabled!"))
		return
	}

	match := pcsmMatch{}

	buf, _ := io.ReadAll(r.Body)
	f.l.Debug("Match from PCSM", "data", string(buf))

	if err := json.Unmarshal(buf, &match); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		f.l.Warn("Error decoding match from PCSM", "error", err, "data", string(buf))
		return
	}

	if err := f.tlm.InsertOnDemandMap(match.toTLM()); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		f.l.Warn("Error inserting on-demand match", "error", err)
		return
	}

	f.l.Info("Remapped field from PCSM", "match", match.Number)
	for _, field := range match.Fields {
		for _, t := range field.Teams {
			f.l.Info("Team Location Change", "field", field.Number, "quadrant", t.Quadrant, "number", t.Number, "team", t.Name)
		}
	}
}
