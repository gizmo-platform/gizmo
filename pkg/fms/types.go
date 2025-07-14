// Package fms encapsulates all the various parts of the Field
// Managent System, its associated configuration logic, and the APIs
// that talk to other systems.
package fms

import (
	"sort"
	"sync"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/http"
)

const (
	// AutomationUser is created on remote systems to allow the
	// FMS to manage them programattically.
	AutomationUser = "gizmo-fms"

	// ViewOnlyUser is created on remote systems to enable
	// debugging and generally make it possible to get into
	// systems.
	ViewOnlyUser = "gizmo-ro"

	// IntegrationPCSM provides API endpoints for the BEST
	// Robotics PCSM to control the match mapping.
	IntegrationPCSM Integration = iota
)

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetFieldForTeam(int) (string, error)
	GetCurrentMapping() (map[int]string, error)
	InsertOnDemandMap(map[int]string) error

	GetStageMapping() (map[int]string, error)
	InsertStageMapping(map[int]string) error
	CommitStagedMap() error
}

type hudVersions struct {
	HardwareVersions string
	FirmwareVersions string
	Bootmodes        string
	DSVersions       string
}

// FMS encapsulates the FMS runnable.
type FMS struct {
	s *http.Server
	c *Config
	l hclog.Logger

	tlm TeamLocationMapper

	swg *sync.WaitGroup
	tpl *pongo2.TemplateSet

	quads []string

	stop           chan struct{}
	connectedDS    map[int]time.Time
	connectedGizmo map[int]time.Time
	connectedMutex *sync.RWMutex
	gizmoMeta      map[int]config.GizmoMeta
	dsMeta         map[int]config.DSMeta
	metaMutex      *sync.RWMutex

	hudVersions hudVersions
}

// Integration is an enum type for things that can talk to the Gizmo
// FMS that may need to be switched on or off.
type Integration int

// IntegrationSlice is used to shadow []int to construct some methods
// that work on the various integrations.
type IntegrationSlice []Integration

// Team maintains information about a team from the perspective of the
// FMS
type Team struct {
	Name     string
	Number   int
	SSID     string
	PSK      string
	VLAN     int
	CIDR     string
	GizmoMAC string
	DSMAC    string
}

// Field contains the information related to each field.
type Field struct {
	ID  int
	IP  string
	MAC string

	Channel string
}

// Config contains all the data that's necessary to setup the FMS and
// manage the network behind it.
type Config struct {
	l hclog.Logger

	// path is where the config was loaded from.
	path string

	// Teams contains the information needed to generate
	// configuration for all teams.
	Teams map[int]*Team

	// Fields contains a list of fields that are configured and
	// managed by the system.
	Fields map[int]*Field

	// FMSMac is the mac address of the FMS itself so that it can
	// have a pinned address
	FMSMac string

	// RadioMode is used to determine which radios are active for
	// the field.  This can be 'NONE', 'FIELD', or 'DS'.  'NONE'
	// predicably disables all radios and is really only useful in
	// games where there is a field tether.  'FIELD' operates
	// using the high-power field radio and causes driver's
	// station radios to be disabled, whereas 'DS' is the inverse
	// causing the field radio to be disabled and remotely
	// controlling the DS radios.
	RadioMode string

	AutoUser string
	AutoPass string
	ViewUser string
	ViewPass string

	// This is the actual "admin" user in RouterOS.  Generally
	// nobody should be logged in as this, but its here anyway so
	// its a known value.
	AdminPass string

	Integrations IntegrationSlice

	InfrastructureVisible bool
	InfrastructureSSID    string
	InfrastructurePSK     string

	// There are cases where fixed DNS servers are desirable.
	FixedDNS []string

	AdvancedBGPAS     int
	AdvancedBGPIP     string
	AdvancedBGPPeerIP string
	AdvancedBGPVLAN   int
}

// SortedTeams returns a list of teams that are sorted by the team
// number.  This makes it easy to visually scan a team list and know
// roughly where they should be.
func (c *Config) SortedTeams() []*Team {
	out := []*Team{}
	for n, t := range c.Teams {
		t.Number = n
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Number < out[j].Number
	})
	return out
}
