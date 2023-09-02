package cmdlets

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
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
	qInitial := []*survey.Question{
		{
			Name: "ServerIP",
			Validate: survey.Required,
			Prompt: &survey.Input{
				Message: "Address of the field server",
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
			},
		},
	}

	cfg := struct {
		ServerIP         string
		FieldCount       int
		AllGamepadsLocal bool
		RemoteGamepads   []string
	}{}

	if err := survey.Ask(qInitial, &cfg); err != nil {
		fmt.Println(err.Error())
		return
	}
	cfg.FieldCount++
	if !cfg.AllGamepadsLocal {

		quadList := []string{}
		for i := 0; i < cfg.FieldCount; i++ {
			for _, c := range []string{"red", "blue", "green", "yellow"} {
				quadList = append(quadList, fmt.Sprintf("field%d:%s", i+1, c))
			}
		}

		qRemoteGamepads := []*survey.Question{
			{
				Name: "RemoteGamepads",
				Prompt: &survey.MultiSelect{
					Message: "Select all remote gamepads",
					Options: quadList,
				},
			},
		}
		if err := survey.Ask(qRemoteGamepads, &cfg); err != nil {
			fmt.Println(err.Error())
			return
		}
	}


	fmt.Println("Your event is configured as follows")
	fmt.Println("")
	fmt.Printf("You have %d field(s)\n", cfg.FieldCount)
	if !cfg.AllGamepadsLocal {
		fmt.Println("You are making use of remote gamepads")
		fmt.Println("The following gamepads are remote from this system")
		for _, g := range cfg.RemoteGamepads {
			fmt.Printf("\t%s\n", g)
		}
	} else {
		fmt.Println("All gamepads are connected directly")
	}
	fmt.Printf("The server's IP is %s\n", cfg.ServerIP)

	confirm := false
	survey.AskOne(&survey.Confirm{Message: "Does everything above look right?"}, &confirm)
}
