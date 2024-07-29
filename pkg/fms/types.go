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
)

// Team maintains information about a team from the perspective of the
// FMS
type Team struct {
	Name string
	SSID string
	PSK  string
	VLAN int
	CIDR string
}

// Field contains the information related to each field.
type Field struct {
	ID  int
	IP  string
	MAC string
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

	AutoUser string
	AutoPass string
	ViewUser string
	ViewPass string

	// This is the actual "admin" user in RouterOS.  Generally
	// nobody should be logged in as this, but its here anyway so
	// its a known value.
	AdminPass string

	InfrastructureVisible bool
	InfrastructureSSID    string
	InfrastructurePSK     string

	AdvancedBGPAS   int
	AdvancedBGPIP   string
	AdvancedBGPVLAN int
}
