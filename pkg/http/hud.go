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

func (hf hudField) Team(quad string) hudQuad {
	switch quad {
	case "red":
		return hf.Red
	case "blue":
		return hf.Blue
	case "green":
		return hf.Green
	case "yellow":
		return hf.Yellow
	}
	return hudQuad{}
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

	m, _ := s.tlm.GetCurrentMapping()

	out := make([]hudField, len(s.fmsConf.Fields))
	for t, f := range m {
		parts := strings.Split(f, ":")
		n, err := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
		if err != nil {
			s.l.Error("Error decoding field number", "error", err)
			continue
		}
		n = n - 1

		fTmp := hudQuad{Team: t}
		s.connectedMutex.RLock()
		_, fTmp.GizmoConnected = s.connectedGizmo[t]
		_, fTmp.DSConnected = s.connectedDS[t]
		s.connectedMutex.RUnlock()

		s.metaMutex.RLock()
		fTmp.GizmoMeta = s.gizmoMeta[t]
		fTmp.DSMeta = s.dsMeta[t]
		s.metaMutex.RUnlock()

		switch parts[1] {
		case "red":
			out[n].Red = fTmp
		case "blue":
			out[n].Blue = fTmp
		case "green":
			out[n].Green = fTmp
		case "yellow":
			out[n].Yellow = fTmp
		}
	}
	ctx["fields"] = out
	ctx["quads"] = []string{"red", "blue", "green", "yellow"}
	ctx["hwversions"] = s.hudVersions.HardwareVersions
	ctx["fwversions"] = s.hudVersions.FirmwareVersions
	ctx["bootmodes"] = s.hudVersions.Bootmodes
	ctx["dsversions"] = s.hudVersions.DSVersions

	s.doTemplate(w, r, "p2/views/field.p2", ctx)
}
