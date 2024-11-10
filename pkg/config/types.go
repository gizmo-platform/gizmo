// Package config contains a convenient structure to pass around
// configuration data.
package config

// Config holds a number of settings that are unique to each driver
// station and gizmo pair.
type Config struct {
	Team             int
	NetSSID          string
	NetPSK           string
	ServerIP         string
}

// DSMeta stores information reported by the Driver's Station metadata
// feed.
type DSMeta struct {
	Version  string
	Bootmode string
}

// GizmoMeta stores information reported by the Gizmo metadata feed.
type GizmoMeta struct {
	HardwareVersion string
	FirmwareVersion string
}
