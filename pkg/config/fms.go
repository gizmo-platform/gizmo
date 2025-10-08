package config

import (
	"encoding/json"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
)

const (
	defconfPath = "/var/lib/gizmo/fms.json"

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

	if c.populateRequiredElements() {
		if err := c.Save(); err != nil {
			l.Warn("Required elements were populated but could not be saved!", "error", err)
		}
		l.Info("Required configuration elements initialized")
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

// Fills in certain elements that should never be null under any
// circumstances.
func (c *FMSConfig) populateRequiredElements() bool {
	needSave := (c.FMSMac == "") || (c.AutoUser == "") || (c.ViewUser == "") ||
		(c.AdminPass == "") || (c.AutoPass == "") || (c.ViewPass == "") ||
		(c.InfrastructureSSID == "") || (c.RadioMode == "") ||
		(c.CompatHardwareVersions == "") || (c.CompatFirmwareVersions == "") ||
		(c.CompatDSBootmodes == "") || (c.CompatDSVersions == "")

	xkcd := xkcdpwgen.NewGenerator()
	xkcd.SetNumWords(3)
	xkcd.SetCapitalize(true)
	xkcd.SetDelimiter("")

	// Check if the FMSMac is unset.  If it is, set it to our own
	// MAC on the premise that we're probably running on the FMS
	// in the default configuration.
	if c.FMSMac == "" {
		eth0, err := netlink.LinkByName("eth0")
		if err != nil {
			c.l.Warn("Could not determine eth0 MAC address", "error", err)
		}
		c.FMSMac = eth0.Attrs().HardwareAddr.String()
	}

	// These are hard-coded elsewhere, and so must always be set
	// to these values unless you REALLY know what you're doing.
	c.AutoUser = AutomationUser
	c.ViewUser = ViewOnlyUser

	// If the passwords are unset, roll new ones.
	if c.AdminPass == "" {
		c.AdminPass = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if c.AutoPass == "" {
		c.AutoPass = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if c.ViewPass == "" {
		c.ViewPass = xkcd.GeneratePasswordString()
	}

	// If the SSID hasn't been set, then set it and the password
	// to an XKCD string.
	if c.InfrastructureSSID == "" {
		c.InfrastructureSSID = "gizmo"
		c.InfrastructurePSK = xkcd.GeneratePasswordString()
	}

	if c.RadioMode == "" {
		c.RadioMode = "FIELD"
	}

	// Set defaults for all the compatibility fields that drive
	// status indicators in the UI.
	if c.CompatHardwareVersions == "" {
		c.CompatHardwareVersions = "GIZMO_V00_R6E,GIZMO_V1_0_R00"
	}
	if c.CompatFirmwareVersions == "" {
		c.CompatFirmwareVersions = "0.1.8"
	}
	if c.CompatDSBootmodes == "" {
		c.CompatDSBootmodes = "RAMDISK"
	}
	if c.CompatDSVersions == "" {
		c.CompatDSVersions = buildinfo.Version // Always accept own version
	}

	return needSave
}
