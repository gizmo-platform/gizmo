package fms

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

func (f *FMS) fieldHUD(w http.ResponseWriter, r *http.Request) {
	ctx := pongo2.Context{}

	m, _ := f.tlm.GetCurrentMapping()

	out := make([]hudField, len(f.c.Fields))
	for team, field := range m {
		parts := strings.Split(field, ":")
		n, err := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
		if err != nil {
			f.l.Error("Error decoding field number", "error", err)
			continue
		}
		n = n - 1

		fTmp := hudQuad{Team: team}
		f.connectedMutex.RLock()
		_, fTmp.GizmoConnected = f.connectedGizmo[team]
		_, fTmp.DSConnected = f.connectedDS[team]
		f.connectedMutex.RUnlock()

		f.metaMutex.RLock()
		fTmp.GizmoMeta = f.gizmoMeta[team]
		fTmp.DSMeta = f.dsMeta[team]
		f.metaMutex.RUnlock()

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
	ctx["hwversions"] = f.hudVersions.HardwareVersions
	ctx["fwversions"] = f.hudVersions.FirmwareVersions
	ctx["bootmodes"] = f.hudVersions.Bootmodes
	ctx["dsversions"] = f.hudVersions.DSVersions
	ctx["shownav"] = false

	f.doTemplate(w, r, "views/field/hud.p2", ctx)
}
