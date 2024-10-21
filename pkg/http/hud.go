package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/flosch/pongo2/v5"
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
	DSConnected       bool
	DSCorrectLocation bool
}

func (s *Server) fieldHUD(w http.ResponseWriter, r *http.Request) {
	ctx := pongo2.Context{}
	clients := s.mq.Clients()
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
		_, fTmp.GizmoConnected = clients[fmt.Sprintf("gizmo-%d", team)]
		_, fTmp.DSConnected = clients[fmt.Sprintf("gizmo-ds%d", team)]
		if fTmp.DSConnected {
			fTmp.DSCorrectLocation = clients[fmt.Sprintf("gizmo-ds%d", team)].CorrectLocation
		}

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

	s.doTemplate(w, r, "p2/views/field.p2", ctx)
}
