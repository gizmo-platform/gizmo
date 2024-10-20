package config

import (
	"bufio"
	"encoding/json"
	"time"

	"github.com/hashicorp/go-hclog"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Server is a binding of the related functions that make up the
// configuration interface for the Gizmo itself.
type Server struct {
	l hclog.Logger
	t *time.Ticker

	once bool

	provider Provider
}

// Option configures the server
type Option func(*Server)

// Provider hands back the configuration for a given Gizmo.  This can
// either be automatic, or with manual intervention, this WILL stall
// the config server if it calls other resources!
type Provider func() Config

// WithLogger sets the logging instance for this config server.
func WithLogger(l hclog.Logger) Option { return func(s *Server) { s.l = l } }

// WithProvider sets up the config provider that will be used by this server.
func WithProvider(p Provider) Option { return func(s *Server) { s.provider = p } }

// WithOneshotMode instructs the configserver to exit after a single
// provisioning cycle.
func WithOneshotMode() Option { return func(s *Server) { s.once = true } }

// NewServer returns the server instance with the options set
func NewServer(opts ...Option) *Server {
	x := &Server{}
	x.l = hclog.NewNullLogger()
	x.t = time.NewTicker(time.Second * 5)

	for _, o := range opts {
		o(x)
	}
	return x
}

// Serve sits and serves forever until shutdown is called.
func (s *Server) Serve() error {
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

func (s *Server) installConfig(name string) {
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
	for scanner.Scan() {
		if scanner.Text() == "GIZMO_REQUEST_CONFIG" {
			break
		}
	}

	if err := json.NewEncoder(port).Encode(s.provider()); err != nil {
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
