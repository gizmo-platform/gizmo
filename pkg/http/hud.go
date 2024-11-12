package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/flosch/pongo2/v5"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

type hudField struct {
	Red    hudQuad
	Blue   hudQuad
	Green  hudQuad
	Yellow hudQuad
}

type hudQuad struct {
	Team              int
	GizmoConnected    bool
	GizmoMeta         config.GizmoMeta
	DSConnected       bool
	DSCorrectLocation bool
	DSMeta            config.DSMeta
}

func (s *Server) fieldHUD(w http.ResponseWriter, r *http.Request) {
	ctx := pongo2.Context{}
	mapping, _ := s.tlm.GetCurrentMapping()

	fields := make(map[int]*hudField)

	for team, quad := range mapping {
		parts := strings.Split(quad, ":")
		fID, _ := strconv.Atoi(strings.Split(parts[0], ":")[0])
		color := strings.ToUpper(parts[1])

		if _, ok := fields[fID]; !ok {
			fields[fID] = &hudField{}
		}

		fTmp := hudQuad{Team: team}
		s.connectedMutex.RLock()
		_, fTmp.GizmoConnected = s.connectedGizmo[team]
		_, fTmp.DSConnected = s.connectedDS[team]
		s.connectedMutex.RUnlock()

		s.metaMutex.RLock()
		fTmp.GizmoMeta = s.gizmoMeta[team]
		fTmp.DSMeta = s.dsMeta[team]
		s.metaMutex.RUnlock()

		switch color {
		case "RED":
			fields[fID].Red = fTmp
		case "BLUE":
			fields[fID].Blue = fTmp
		case "GREEN":
			fields[fID].Green = fTmp
		case "YELLOW":
			fields[fID].Yellow = fTmp
		}

	}
	ctx["fields"] = fields
	ctx["hwversions"] = s.hudVersions.HardwareVersions
	ctx["fwversions"] = s.hudVersions.FirmwareVersions
	ctx["bootmodes"] = s.hudVersions.Bootmodes
	ctx["dsversions"] = s.hudVersions.DSVersions

	s.doTemplate(w, r, "p2/views/field.p2", ctx)
}
