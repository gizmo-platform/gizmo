// Package config contains a convenient structure to pass around
// configuration data.
package config

import (
	"strings"

	"github.com/hashicorp/go-hclog"
)

// GSSConfig holds a number of settings that are unique to each driver
// station and gizmo pair.
type GSSConfig struct {
	Team     int
	NetSSID  string
	NetPSK   string
	ServerIP string
	FieldIP  string
}

// DSMeta stores information reported by the Driver's Station metadata
// feed.
type DSMeta struct {
	Version  string
	Bootmode string
}

// VersionOK compares the version against a string containing all
// permitted versions.
func (d DSMeta) VersionOK(vList string) bool {
	if d.Version == "" {
		return false
	}
	return strings.Contains(vList, d.Version)
}

// BootmodeOK compares if the boot mode is in the list of accepted
// bootmodes.
func (d DSMeta) BootmodeOK(bList string) bool {
	if d.Bootmode == "" {
		return false
	}
	return strings.Contains(bList, d.Bootmode)
}

// GizmoMeta stores information reported by the Gizmo metadata feed.
type GizmoMeta struct {
	HardwareVersion string
	FirmwareVersion string
}

// HWVersionOK checks if the hardware version is in the list of known
// hardware versions.
func (g GizmoMeta) HWVersionOK(hList string) bool {
	if g.HardwareVersion == "" {
		return false
	}
	return strings.Contains(hList, g.HardwareVersion)
}

// FWVersionOK checks if the firmware version is in the list of known
// firmware versions.
func (g GizmoMeta) FWVersionOK(fList string) bool {
	if g.FirmwareVersion == "" {
		return false
	}
	return strings.Contains(fList, g.FirmwareVersion)
}

// Field contains the information related to each field.
type Field struct {
	ID  int
	IP  string
	MAC string

	Channel string
}

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

// FMSConfig contains all the data that's necessary to setup the FMS and
// manage the network behind it.
type FMSConfig struct {
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

// Integration is an enum type for things that can talk to the Gizmo
// FMS that may need to be switched on or off.
type Integration int

// IntegrationSlice is used to shadow []int to construct some methods
// that work on the various integrations.
type IntegrationSlice []Integration

const (
	// IntegrationPCSM provides API endpoints for the BEST
	// Robotics PCSM to control the match mapping.
	IntegrationPCSM Integration = iota
)
