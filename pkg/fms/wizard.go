//go:build linux

package fms

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
	"github.com/vishvananda/netlink"
)

// ws binds together all the steps required to configure the FMS
type ws struct {
	c *Config
}

// WizardSurvey runs a step by step config workflow to gather all the
// information required to generate the software configuration for the
// FMS.
func WizardSurvey() (*Config, error) {
	w := new(ws)
	w.c = new(Config)
	w.initCfg()
	if err := w.loadTeams(); err != nil {
		return nil, err
	}

	if err := w.setFields(); err != nil {
		return nil, err
	}

	if err := w.setInfraNetwork(); err != nil {
		return nil, err
	}

	if err := w.setFMSMac(); err != nil {
		return nil, err
	}

	advanced := false
	qAdvanced := &survey.Confirm{
		Message: "Configure really advanced network features?",
	}

	if err := survey.AskOne(qAdvanced, &advanced); err != nil {
		return nil, err
	}

	if advanced {
		if err := w.advancedNetCfg(); err != nil {
			return nil, err
		}
	}

	return w.c, nil
}

// WizardChangeRoster is used to change the roster in an FMS Config
// that already exists.  Teams will be loaded then reconciled.
// Existing teams can have name updated, but wireless parameters will
// not be changed.
func WizardChangeRoster(c *Config) (*Config, error) {
	w := new(ws)
	w.c = new(Config)
	w.initCfg()
	if err := w.loadTeams(); err != nil {
		return nil, err
	}

	// Add any team that isn't in the existing import, and update
	// the VLAN and name for any team that is present.
	seen := make(map[int]struct{})
	for num, team := range w.c.Teams {
		seen[num] = struct{}{}
		if _, exists := c.Teams[num]; exists {
			c.Teams[num].Name = w.c.Teams[num].Name
			c.Teams[num].VLAN = w.c.Teams[num].VLAN
		} else {
			c.Teams[num] = team
		}
	}

	// Delete any team that isn't present in the new list.
	for id := range c.Teams {
		if _, present := seen[id]; !present {
			delete(c.Teams, id)
		}
	}

	return c, nil
}

func (w *ws) initCfg() {
	w.c.Teams = make(map[int]*Team)
	w.c.Fields = make(map[int]*Field)
}

func (w *ws) advancedNetCfg() error {
	prompts := []*survey.Question{
		{
			Name:     "AdvancedBGPAS",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "ASN",
				Default: "64512",
			},
		},
		{
			Name:     "AdvancedBGPIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Peer IP",
				Default: "169.254.255.100/24",
			},
		},
		{
			Name:     "AdvancedBGPVLAN",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Peer VLAN",
				Default: "101",
			},
		},
	}

	return survey.Ask(prompts, w.c)
}

func (w *ws) setFMSMac() error {
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}

	prompt := &survey.Input{
		Message: "MAC Address of the FMS",
		Default: eth0.Attrs().HardwareAddr.String(),
	}

	return survey.AskOne(prompt, &w.c.FMSMac)
}

func (w *ws) setInfraNetwork() error {
	xkcd := xkcdpwgen.NewGenerator()
	xkcd.SetNumWords(3)
	xkcd.SetCapitalize(true)
	xkcd.SetDelimiter("")

	prompts := []*survey.Question{
		{
			Name:     "InfrastructureVisible",
			Validate: survey.Required,
			Prompt: &survey.Confirm{
				Message: "Make infrastructure network visible.",
				Default: true,
			},
		},
		{
			Name:     "InfrastructureSSID",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Infrastructure network SSID",
				Default: "gizmo",
			},
		},
		{
			Name:     "InfrastructurePSK",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Infrastructure network PSK",
				Default: xkcd.GeneratePasswordString(),
			},
		},
	}

	return survey.Ask(prompts, w.c)
}

func (w *ws) setFields() error {
	numFields := 0
	prompt := &survey.Select{
		Message: "Select the number of fields present",
		Options: []string{"1", "2", "3"},
		Default: "1",
	}

	if err := survey.AskOne(prompt, &numFields); err != nil {
		return err
	}

	for i := 0; i <= numFields; i++ {
		mac := ""
		fieldPrompt := &survey.Input{
			Message: fmt.Sprintf("Input the MAC address for ether1 for field %d (label on the bottom)", i+1),
		}
		if err := survey.AskOne(fieldPrompt, &mac); err != nil {
			return err
		}

		w.c.Fields[i] = &Field{
			ID:  i + 1,
			IP:  fmt.Sprintf("100.64.0.%d", 10+i),
			MAC: mac,
		}
	}

	w.c.AutoUser = AutomationUser
	w.c.AutoPass = strings.ReplaceAll(uuid.New().String(), "-", "")

	w.c.AdminPass = strings.ReplaceAll(uuid.New().String(), "-", "")

	xkcd := xkcdpwgen.NewGenerator()
	xkcd.SetNumWords(3)
	xkcd.SetCapitalize(true)
	xkcd.SetDelimiter("")
	pPrompt := &survey.Input{
		Message: fmt.Sprintf("Read-only user password (username: %s)", ViewOnlyUser),
		Default: xkcd.GeneratePasswordString(),
	}
	w.c.ViewUser = ViewOnlyUser
	return survey.AskOne(pPrompt, &w.c.ViewPass)
}

func (w *ws) loadTeams() error {
	teamPath := ""
	prompt := &survey.Input{
		Message: "Specify teams CSV file:",
		Suggest: func(toComplete string) []string {
			files, _ := filepath.Glob(toComplete + "*.csv")
			return files
		},
	}

	if err := survey.AskOne(prompt, &teamPath); err != nil {
		return err
	}

	f, err := os.Open(teamPath)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := loadTeams(f)
	if err != nil {
		return err
	}
	w.c.Teams = t

	confirm := false
	cPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Loaded %d teams, does this look right?", len(w.c.Teams)),
	}

	if err := survey.AskOne(cPrompt, &confirm); err != nil {
		return err
	}

	if !confirm {
		return errors.New("configuration cancelled by user")
	}

	return nil
}
