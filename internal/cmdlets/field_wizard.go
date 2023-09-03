package cmdlets

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-sockaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fieldWizardCommand = &cobra.Command{
		Use:   "wizard",
		Short: "wizard provides guided field setup",
		Long:  fieldWizardCmdLongDocs,
		Run:   fieldWizardCmdRun,
	}

	fieldWizardCmdLongDocs = `wizard provides an end to end guided
experience for setting up your field(s).`
)

func init() {
	fieldCmd.AddCommand(fieldWizardCommand)
}

func fieldWizardCmdRun(c *cobra.Command, args []string) {
	// We throw away this error because the worst case is the
	// default isn't set.
	lAddr, _ := sockaddr.GetPrivateIP()

	qInitial := []*survey.Question{
		{
			Name:     "ServerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Address of the field server",
				Default: lAddr,
			},
		},
		{
			Name: "FieldCount",
			Prompt: &survey.Select{
				Message: "Select the number of fields present",
				Options: []string{"1", "2", "3"},
				Default: "1",
			},
		},
		{
			Name: "AllGamepadsLocal",
			Prompt: &survey.Confirm{
				Message: "All gamepads are connected directly via USB",
				Default: true,
			},
		},
	}

	cfg := struct {
		ServerIP         string
		FieldCount       int
		AllGamepadsLocal bool
		Quads            []string
		LocalGamepads    map[string]struct{}
	}{}
	cfg.LocalGamepads = make(map[string]struct{})

	if err := survey.Ask(qInitial, &cfg); err != nil {
		fmt.Println(err.Error())
		return
	}
	cfg.FieldCount++
	for i := 0; i < cfg.FieldCount; i++ {
		for _, c := range []string{"red", "blue", "green", "yellow"} {
			cfg.Quads = append(cfg.Quads, fmt.Sprintf("field%d:%s", i+1, c))
		}
	}
	if !cfg.AllGamepadsLocal {
		qLocalGamepads := &survey.MultiSelect{
			Message: "Select all local gamepads",
			Options: cfg.Quads,
		}
		gsScratch := []string{}
		if err := survey.AskOne(qLocalGamepads, &gsScratch); err != nil {
			fmt.Println(err.Error())
			return
		}
		for _, q := range gsScratch {
			cfg.LocalGamepads[q] = struct{}{}
		}
	}

	fmt.Println("Your event is configured as follows")
	fmt.Println("")
	fmt.Printf("You have %d field(s)\n", cfg.FieldCount)
	if !cfg.AllGamepadsLocal {
		fmt.Println("You are making use of remote gamepads")
		fmt.Println("The following gamepads are directly attached system")
		for g := range cfg.LocalGamepads {
			fmt.Printf("\t%s\n", g)
		}
	} else {
		fmt.Println("All gamepads are connected directly")
	}
	fmt.Printf("The server's IP is %s\n", cfg.ServerIP)

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Does everything above look right?"}, &confirm)

	if !confirm {
		fmt.Println("Bailing out!  Re-run the wizard and correct any errors!")
		os.Exit(1)
	}

	viper.Set("server.address", cfg.ServerIP)
	quads := make([]quad, len(cfg.Quads))
	for id, quadName := range cfg.Quads {
		_, local := cfg.LocalGamepads[quadName]
		quads[id] = quad{Name: quadName, Gamepad: id, Local: local}
	}
	viper.Set("quadrants", quads)
	viper.SetConfigType("yaml")
	viper.WriteConfigAs("config.yml")
}
