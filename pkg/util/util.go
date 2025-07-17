package util

import (
	"net"
)

// NumberToMAC encodes the specified team number as a locally
// administered MAC addresses with the device index as specified.
func NumberToMAC(team, index int) net.HardwareAddr {
	d1 := (team / 1000) * 16
	d2 := (team % 1000) / 100
	d3 := (team % 1000) % 100 / 10 * 16
	d4 := (team % 1000) % 100 % 10

	return []byte{0x02, 0x00, 0x00, byte(d1 + d2), byte(d3 + d4), byte(index)}
}
