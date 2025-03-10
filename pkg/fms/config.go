package fms

import (
	"encoding/json"
	"os"

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
	c.Teams = make(map[int]*Team)
	c.Fields = make(map[int]*Field)
	c.path = os.Getenv("GIZMO_FMS_CONFIG")
	if c.path == "" {
		c.path = defconfPath
	}

	c.l = l.Named("config")
	if err := c.Load(c.path); err != nil {
		return c, err
	}

	return c, nil
}

// Load loads a config file from the given path on disk.
func (c *Config) Load(path string) error {
	f, err := os.Open(path)
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
