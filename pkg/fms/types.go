// Package fms encapsulates all the various parts of the Field
// Managent System, its associated configuration logic, and the APIs
// that talk to other systems.
package fms

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
