// Package config contains a convenient structure to pass around
// configuration data.
package config

// Config holds a number of settings that are unique to each driver
// station and gizmo pair.
type Config struct {
	Team             int
	UseDriverStation bool
	UseExtNet        bool
	NetSSID          string
	NetPSK           string
	ServerIP         string
}
