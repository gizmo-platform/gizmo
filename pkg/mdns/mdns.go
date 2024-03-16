package mdns

import (
	"net"

	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/mdns"
)

// Server wraps the underlying mDNS implementation to provide a
// simplified interface.
type Server struct {
	*mdns.Server
}

// NewServer instantiates a new mDNS server.
func NewServer(name string) (*Server, error) {
	lAddr, _ := sockaddr.GetPrivateIP()

	info := []string{"Gizmo Field Management System"}
	service, err := mdns.NewMDNSService("gizmo"+name, "_gizmo"+name+"._tcp", "", "gizmo-mqtt.local.", 1883, []net.IP{net.ParseIP(lAddr)}, info)
	if err != nil {
		return nil, err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}

	return &Server{server}, nil
}
