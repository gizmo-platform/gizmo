package config

import (
	"encoding/json"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/hashicorp/go-hclog"
)

const (
	defconfPath = "/etc/gizmo/fms.json"

	// AutomationUser is created on remote systems to allow the
	// FMS to manage them programattically.
	AutomationUser = "gizmo-fms"

	// ViewOnlyUser is created on remote systems to enable
	// debugging and generally make it possible to get into
	// systems.
	ViewOnlyUser = "gizmo-ro"
)

// NewFMSConfig returns a new configuration instance, that has been
// loaded from the default path if possible.
func NewFMSConfig(l hclog.Logger) (*FMSConfig, error) {
	if l == nil {
		l = hclog.NewNullLogger()
	}

	c := new(FMSConfig)
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
func (c *FMSConfig) Load() error {
	f, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(c)
}

// Save persists a config to the named path, creating it if necessary.
func (c *FMSConfig) Save() error {
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(c)
}

// SortedTeams returns a list of teams that are sorted by the team
// number.  This makes it easy to visually scan a team list and know
// roughly where they should be.
func (c *FMSConfig) SortedTeams() []*Team {
	out := []*Team{}
	for n, t := range c.Teams {
		t.Number = n
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Number < out[j].Number
	})
	return out
}
