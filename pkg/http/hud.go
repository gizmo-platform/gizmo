package http

import (
	"fmt"
	"net/http"

	"github.com/flosch/pongo2/v5"
)

type hudTableRow struct {
	Number            int
	Gizmo             bool
	DS                bool
	DSCorrectLocation bool
}

func (s *Server) fieldHUD(w http.ResponseWriter, r *http.Request) {
	ctx := pongo2.Context{}
	clients := s.mq.Clients()
	mapping, _ := s.tlm.GetCurrentMapping()

	table := make(map[string]hudTableRow)

	for team, quad := range mapping {
		r := hudTableRow{Number: team}

		_, r.Gizmo = clients[fmt.Sprintf("gizmo-%d", team)]
		_, r.DS = clients[fmt.Sprintf("gizmo-ds%d", team)]
		if r.DS {
			r.DSCorrectLocation = clients[fmt.Sprintf("gizmo-ds%d", team)].CorrectLocation
		}

		table[quad] = r
	}
	ctx["hudTable"] = table

	s.doTemplate(w, r, "p2/views/field.p2", ctx)
}
