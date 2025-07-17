package config

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/gizmo-platform/gizmo/pkg/util"
)

// LoadTeams returns a map of team numbers to team structs, and is
// meant to allow loading from any reader, usually a file..
func LoadTeams(fr io.Reader) (map[int]*Team, error) {
	teams := make(map[int]*Team)
	r := csv.NewReader(fr)
	var header []string
	vlan := 500
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header == nil {
			header = record
			for col := range header {
				header[col] = strings.ReplaceAll(header[col], "Team Name", "Name")
				header[col] = strings.ReplaceAll(header[col], "Team Number", "Number")
				header[col] = strings.ReplaceAll(header[col], "Hub Name", "Hub")
			}
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			num, err := strconv.Atoi(dict["Number"])
			if err != nil {
				return nil, fmt.Errorf("bad team number: %s %s", dict["Name"], dict["Number"])
			}
			teams[num] = &Team{
				VLAN:     vlan,
				Name:     dict["Name"],
				SSID:     strings.ReplaceAll(uuid.New().String(), "-", ""),
				PSK:      strings.ReplaceAll(uuid.New().String(), "-", ""),
				CIDR:     fmt.Sprintf("10.%d.%d.0/24", int(num/100), num%100),
				GizmoMAC: util.NumberToMAC(num, 0).String(),
				DSMAC:    util.NumberToMAC(num, 1).String(),
			}
			vlan++
		}
	}
	return teams, nil
}
