package fms

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
)

const (
	defconfPath = "/etc/gizmo/fms.json"
)

// NewConfig returns a new configuration instance, that has been
// loaded from the default path if possible.
func NewConfig(l hclog.Logger) (*Config, error) {
	if l == nil {
		l = hclog.NewNullLogger()
	}

	c := new(Config)
	c.l = l.Named("config")
	c.Teams = make(map[int]*Team)
	c.Fields = make(map[int]*Field)
	c.path = os.Getenv("GIZMO_FMS_CONFIG")
	if c.path == "" {
		c.path = defconfPath
	}
	c.l.Debug("config path set", "path", c.path)

	if err := c.Load(); err != nil {
		return c, err
	}

	// If we made it here then the config loaded, so we'll go
	// ahead and setup the reloader so that its possible to update
	// the config again later.
	rChan := make(chan os.Signal, 1)
	signal.Notify(rChan, syscall.SIGHUP)

	go func() {
		for {
			<-rChan
			c.l.Debug("Config reload requested")
			if err := c.Load(); err != nil {
				c.l.Warn("Error reloading config", "error", err)
			}
		}
	}()

	return c, nil
}

// Load loads a config file from the given path on disk.
func (c *Config) Load() error {
	f, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(c)
}

// Save persists a config to the named path, creating it if necessary.
func (c *Config) Save() error {
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(c)
}
