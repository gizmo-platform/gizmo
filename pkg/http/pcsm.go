package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gizmo-platform/gizmo/pkg/fms"
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

	for _, f := range p.Fields {
		for _, t := range f.Teams {
			if t.Number == 0 {
				continue
			}
			out[t.Number] = fmt.Sprintf("field%d:%s", f.Number, strings.ToLower(t.Quadrant))
		}
	}

	return out
}

func (s *Server) remapTeamsPCSM(w http.ResponseWriter, r *http.Request) {
	if !s.fmsConf.Integrations.Enabled(fms.IntegrationPCSM) {
		w.WriteHeader(http.StatusPreconditionFailed)
		w.Write([]byte("Integration is not enabled!"))
		return
	}

	match := pcsmMatch{}

	buf, _ := io.ReadAll(r.Body)
	s.l.Debug("Match from PCSM", "data", string(buf))

	if err := json.Unmarshal(buf, &match); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.l.Warn("Error decoding match from PCSM", "error", err, "data", string(buf))
		return
	}

	if err := s.tlm.InsertOnDemandMap(match.toTLM()); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.l.Warn("Error inserting on-demand match", "error", err)
		return
	}

	s.l.Info("Remapped field from PCSM", "match", match.Number)
	for _, f := range match.Fields {
		for _, t := range f.Teams {
			s.l.Info("Team Location Change", "field", f.Number, "quadrant", t.Quadrant, "number", t.Number, "team", t.Name)
		}
	}
}
