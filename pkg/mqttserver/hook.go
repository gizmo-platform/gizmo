package mqttserver

import (
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// GizmoHook handles all the custom logic for Gizmo
type GizmoHook struct {
	mqtt.HookBase

	l hclog.Logger
}

func newHook(l hclog.Logger) *GizmoHook {
	gh := new(GizmoHook)
	gh.l = l
	return gh
}

// Provides flags which methods the server will invoke this hook for.
// Adding or removing methods in this file requires updating this
// value!
func (gh *GizmoHook) Provides(b byte) bool {
	provides := map[byte]struct{}{
		mqtt.OnACLCheck:            struct{}{},
		mqtt.OnConnectAuthenticate: struct{}{},
		mqtt.OnConnect:             struct{}{},
		mqtt.OnDisconnect:          struct{}{},
		mqtt.OnSessionEstablished:  struct{}{},
		mqtt.OnStarted:             struct{}{},
		mqtt.OnSubscribed:          struct{}{},
	}
	_, ok := provides[b]
	return ok
}

// ID identifies this hook in the listing.
func (gh *GizmoHook) ID() string {
	return "GizmoHook"
}

// OnStarted happens after the listeners are bound and the server is
// ready to process connections.
func (gh *GizmoHook) OnStarted() {
	gh.l.Info("Ready for connections")
}

// OnSessionEstablished happens after a client is completely connected
// and ready to send and receive data.
func (gh *GizmoHook) OnSessionEstablished(cl *mqtt.Client, pk packets.Packet) {
	if strings.HasPrefix(cl.ID, "gizmo") {
		expected := teamNumberFromClient(cl)
		gh.l.Info("Client Connected", "client", cl.ID, "expected", expected)
	}
}

// OnConnect fires when a client connects, and we use this to forcibly
// clear all state for clients connecting to the server.
func (gh *GizmoHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	gh.l.Debug("Client Connect", "client", cl.ID)
	cl.ClearInflights()
	return nil
}

// OnDisconnect fires when a client is disconnected for any reason.
func (gh *GizmoHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	gh.l.Info("Client Disconnected", "client", cl.ID, "expired", expire)
}

// OnConnectAuthenticate allows anyone to connect, but what they can
// then do is pretty heavily limited by the OnACLCheck below.
func (gh *GizmoHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	return true
}

// OnACLCheck gets called to work out if a client should be allowed to
// do things or not.  The first check that we make is if the client is
// in either 127.0.0.0/8 (the server itself) or 100.64.0.0/24 (the FMS
// netblock).  If either of these is true than the result is returned
// immediately as success.  Otherwise the actual team number is
// checked to make sure it corresponds to the one in the topic.
func (gh *GizmoHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	host, _, err := net.SplitHostPort(cl.Net.Remote)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	_, fms, _ := net.ParseCIDR("100.64.0.0/24")

	if ip.IsLoopback() || fms.Contains(ip) {
		return true
	}

	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return false
	}

	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	expectedTeam := teamNumberFromClient(cl)
	approved := num == expectedTeam
	return approved
}

// OnSubscribed logs subscriptions as they come in for a given client.
// Useful for debugging and normally a noop.
func (gh *GizmoHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	s := cl.State.Subscriptions.GetAll()
	subs := []string{}
	for k := range s {
		subs = append(subs, k)
	}
	gh.l.Debug("Subscribed", "client", cl.ID, "subscriptions", subs)
}
