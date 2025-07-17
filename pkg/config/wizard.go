//go:build linux

package config

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

// WizardSurvey runs a step by step config workflow to gather all the
// information required to generate the software configuration for the
// FMS.
func (c *FMSConfig) WizardSurvey(wouldOverwrite bool) error {
	if err := c.bigScaryOverwriteWarning(wouldOverwrite); err != nil {
		return err
	}

	if err := c.loadTeams(); err != nil {
		return err
	}

	if err := c.setRadioMode(); err != nil {
		return err
	}

	if err := c.setFields(); err != nil {
		return err
	}

	if err := c.setInfraNetwork(); err != nil {
		return err
	}

	if err := c.setFMSMac(); err != nil {
		return err
	}

	if err := c.setIntegrations(); err != nil {
		return err
	}

	advanced := false
	qAdvanced := &survey.Confirm{
		Message: "Configure really advanced network features?",
	}

	if err := survey.AskOne(qAdvanced, &advanced); err != nil {
		return err
	}

	if advanced {
		if err := c.advancedNetCfg(); err != nil {
			return err
		}
	}

	return nil
}

// WizardChangeRoster is used to change the roster in an FMS Config
// that already exists.  Teams will be loaded then reconciled.
// Existing teams can have name updated, but wireless parameters will
// not be changed.
func (c *FMSConfig) WizardChangeRoster() error {
	if err := c.loadTeams(); err != nil {
		return err
	}

	// Add any team that isn't in the existing import, and update
	// the VLAN and name for any team that is present.
	seen := make(map[int]struct{})
	for num, team := range c.Teams {
		seen[num] = struct{}{}
		if _, exists := c.Teams[num]; exists {
			c.Teams[num].Name = c.Teams[num].Name
			c.Teams[num].VLAN = c.Teams[num].VLAN
			c.Teams[num].GizmoMAC = c.Teams[num].GizmoMAC
			c.Teams[num].DSMAC = c.Teams[num].DSMAC
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

	return nil
}

// WizardChangeChannels is used to reconfigure the channels each field
// is assigned to.  This function duplicates some of the config in the
// field configuration step, but this is to avoid any possibility of
// creating or destroying a field.
func (c *FMSConfig) WizardChangeChannels() error {
	for i := range c.Fields {
		channelPrompt := &survey.Select{
			Message: "Select the channel to pin this field to.  You can change this later.",
			Options: []string{"Auto", "1", "6", "11"},
			Default: func() string {
				if c.Fields[i].Channel != "" {
					return c.Fields[i].Channel
				}
				return "Auto"
			}(),
		}
		if err := survey.AskOne(channelPrompt, &c.Fields[i].Channel); err != nil {
			return err
		}
	}

	return nil
}

// WizardChangeRadioMode is used to reconfigure the radio mode after
// it has been initially setup.
func (c *FMSConfig) WizardChangeRadioMode() error {
	return c.setRadioMode()
}

// WizardChangeIntegrations can be used to reconfigure what
// integrations are enabled.
func (c *FMSConfig) WizardChangeIntegrations() error {
	return c.setIntegrations()
}

func (c *FMSConfig) bigScaryOverwriteWarning(wouldOverwrite bool) error {
	if !wouldOverwrite {
		return nil
	}

	qOverwrite := &survey.Confirm{
		Message: "Overwrite existing config?  You will have to re-flash all devices if you answer yes!.",
	}

	fmt.Println("                         BIG SCARY WARNING")
	fmt.Println()
	fmt.Println("This question asks you to confirm that you are okay wiping your old")
	fmt.Println("configuration, which will require you to reconfigure all your network")
	fmt.Println("devices.  This is normal if you're resetting between seasons or major")
	fmt.Println("events, but is not normal within a single competition season or")
	fmt.Println("event.")

	if err := survey.AskOne(qOverwrite, &wouldOverwrite); err != nil {
		return err
	}
	if !wouldOverwrite {
		return fmt.Errorf("configuration canceled")
	}

	return nil
}

func (c *FMSConfig) advancedNetCfg() error {
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
		{
			Name:     "AdvancedBGPPeerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Peer IP (no subnet)",
				Default: "169.254.255.8",
			},
		},
	}

	return survey.Ask(prompts, c)
}

func (c *FMSConfig) setFMSMac() error {
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}

	prompt := &survey.Input{
		Message: "MAC Address of the FMS",
		Default: eth0.Attrs().HardwareAddr.String(),
	}

	return survey.AskOne(prompt, &c.FMSMac)
}

func (c *FMSConfig) setRadioMode() error {
	prompt := &survey.Select{
		Message: "Select Radio Mode",
		Default: func() string {
			if c.RadioMode == "" {
				return "FIELD"
			}
			return c.RadioMode
		}(),
		Options: []string{"NONE", "FIELD", "DS"},
	}

	return survey.AskOne(prompt, &c.RadioMode)
}

func (c *FMSConfig) setInfraNetwork() error {
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
			Name: "InfrastructurePSK",
			Validate: survey.ComposeValidators(
				survey.MinLength(8),
				survey.MaxLength(63),
				survey.Required,
			),
			Prompt: &survey.Input{
				Message: "Infrastructure network PSK",
				Default: xkcd.GeneratePasswordString(),
			},
		},
	}

	return survey.Ask(prompts, c)
}

func (c *FMSConfig) setFields() error {
	numFields := 0
	prompt := &survey.Select{
		Message: "Select the number of fields present",
		Options: []string{"1", "2", "3", "4", "5", "6"},
		Default: "1",
	}

	if err := survey.AskOne(prompt, &numFields); err != nil {
		return err
	}

	for i := 0; i <= numFields; i++ {
		mac := ""
		channel := ""
		fieldPrompt := &survey.Input{
			Message: fmt.Sprintf("Input the MAC address for ether1 for field %d (label on the bottom)", i+1),
		}
		if err := survey.AskOne(fieldPrompt, &mac); err != nil {
			return err
		}

		channelPrompt := &survey.Select{
			Message: "Select the channel to pin this field to.  You can change this later.",
			Options: []string{"Auto", "1", "6", "11"},
		}
		if err := survey.AskOne(channelPrompt, &channel); err != nil {
			return err
		}

		c.Fields[i] = &Field{
			ID:      i + 1,
			IP:      fmt.Sprintf("100.64.0.%d", 10+i),
			MAC:     mac,
			Channel: channel,
		}
	}

	c.AutoUser = AutomationUser
	c.AutoPass = strings.ReplaceAll(uuid.New().String(), "-", "")

	c.AdminPass = strings.ReplaceAll(uuid.New().String(), "-", "")

	xkcd := xkcdpwgen.NewGenerator()
	xkcd.SetNumWords(3)
	xkcd.SetCapitalize(true)
	xkcd.SetDelimiter("")
	pPrompt := &survey.Input{
		Message: fmt.Sprintf("Read-only user password (username: %s)", ViewOnlyUser),
		Default: xkcd.GeneratePasswordString(),
	}
	c.ViewUser = ViewOnlyUser
	return survey.AskOne(pPrompt, &c.ViewPass)
}

func (c *FMSConfig) setIntegrations() error {
	prompt := &survey.MultiSelect{
		Message: "Select Integrations",
		Options: allIntegrations.ToStrings(),
		Default: c.Integrations.ToStrings(),
	}

	integrations := []string{}
	if err := survey.AskOne(prompt, &integrations); err != nil {
		return nil
	}

	c.Integrations = IntegrationsFromStrings(integrations)
	return nil
}

func (c *FMSConfig) loadTeams() error {
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

	t, err := LoadTeams(f)
	if err != nil {
		return err
	}
	c.Teams = t

	confirm := false
	cPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Loaded %d teams, does this look right?", len(c.Teams)),
	}

	if err := survey.AskOne(cPrompt, &confirm); err != nil {
		return err
	}

	if !confirm {
		return errors.New("configuration cancelled by user")
	}

	return nil
}
