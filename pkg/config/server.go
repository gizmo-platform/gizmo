package config

import (
	"bufio"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// GSSServer is a binding of the related functions that make up the
// configuration interface for the Gizmo itself.
type GSSServer struct {
	l hclog.Logger
	t *time.Ticker

	once bool

	provider Provider
}

// Option configures the server
type Option func(*GSSServer)

// Provider hands back the configuration for a given Gizmo.  This can
// either be automatic, or with manual intervention, this WILL stall
// the config server if it calls other resources!
type Provider func(team int) GSSConfig

// WithLogger sets the logging instance for this config server.
func WithLogger(l hclog.Logger) Option { return func(s *GSSServer) { s.l = l } }

// WithProvider sets up the config provider that will be used by this server.
func WithProvider(p Provider) Option { return func(s *GSSServer) { s.provider = p } }

// WithOneshotMode instructs the configserver to exit after a single
// provisioning cycle.
func WithOneshotMode() Option { return func(s *GSSServer) { s.once = true } }

// NewGSSServer returns the server instance with the options set
func NewGSSServer(opts ...Option) *GSSServer {
	x := &GSSServer{}
	x.l = hclog.NewNullLogger()
	x.t = time.NewTicker(time.Second * 5)

	for _, o := range opts {
		o(x)
	}
	return x
}

// Serve sits and serves forever until shutdown is called.
func (s *GSSServer) Serve() error {
	for range s.t.C {
		ports, err := enumerator.GetDetailedPortsList()
		if err != nil {
			return err
		}
		for _, port := range ports {
			s.l.Info("Found a port!", "port", port.Name)
			if !port.IsUSB {
				// We know the Gizmo must be connected via USB
				continue
			}
			if port.VID == "2e8a" && port.PID == "f00a" {
				s.installConfig(port.Name)
			}
			if s.once {
				s.t.Stop()
				return nil
			}
		}
	}
	return nil
}

func (s *GSSServer) installConfig(name string) {
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(name, mode)
	if err != nil {
		s.l.Error("Could not open port", "port", name, "error", err)
		return
	}
	defer port.Close()

	scanner := bufio.NewScanner(bufio.NewReader(port))
	team := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "GIZMO_REQUEST_CONFIG") {
			parts := strings.Fields(scanner.Text())
			if len(parts) == 2 {
				// The second field should be the team
				// number, we need to cast it and set
				// that as the current team number.
				team, err = strconv.Atoi(parts[1])
				if err != nil {
					s.l.Warn("Could not parse team number", "error", err)
				}
			}
			break
		}
	}

	if err := json.NewEncoder(port).Encode(s.provider(team)); err != nil {
		s.l.Error("Error serializing configuration", "error", err)
		return
	}

	if err := port.Drain(); err != nil {
		s.l.Error("Error draining port", "port", name, "error", err)
		return
	}
	s.l.Info("Upload complete")
	if !s.once {
		time.Sleep(time.Second * 5)
	}
}
