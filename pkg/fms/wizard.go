package fms

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
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
				Default: "169.254.2.2",
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
		w.c.Fields[i] = &Field{
			ID: i + 1,
			IP: fmt.Sprintf("10.0.0.%d", 10+i),
		}
	}

	w.c.AutoUser = AutomationUser
	w.c.AutoPass = strings.ReplaceAll(uuid.New().String(), "-", "")

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

	r := csv.NewReader(f)
	var header []string
	vlan := 100
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header == nil {
			header = record
			for col := range header {
				header[col] = strings.ReplaceAll(header[col], "Team Name", "Name")
				header[col] = strings.ReplaceAll(header[col], "Team Number", "Number")
				header[col] = strings.ReplaceAll(header[col], "Hub Name", "Hub")
			}
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			num, err := strconv.Atoi(dict["Number"])
			if err != nil {
				return fmt.Errorf("bad team number: %s %s", dict["Name"], dict["Number"])
			}
			w.c.Teams[num] = &Team{
				VLAN: vlan,
				Name: dict["Name"],
				SSID: strings.ReplaceAll(uuid.New().String(), "-", ""),
				PSK:  strings.ReplaceAll(uuid.New().String(), "-", ""),
				CIDR: fmt.Sprintf("10.%d.%d.0/24", int(num/100), num%100),
			}
			vlan++
		}
	}

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
