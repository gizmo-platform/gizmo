package cmdlets

import (
	"fmt"
	"os"
	"sort"

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
		GamepadPushers   map[string]string
	}{}
	cfg.GamepadPushers = make(map[string]string)

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
		// Setup a set of quads that need to be assigned,
		// using a map so we can delete() out of it as they
		// are assigned.
		quadScratch := make(map[string]struct{})
		for _, q := range cfg.Quads {
			quadScratch[q] = struct{}{}
		}

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
			cfg.GamepadPushers[q] = "self"
			delete(quadScratch, q)
		}

		for len(quadScratch) > 0 {
			qPusherName := &survey.Input{
				Message: "Name of remote pusher",
			}
			pusherName := ""
			if err := survey.AskOne(qPusherName, &pusherName); err != nil {
				fmt.Println(err.Error)
				return
			}

			pusherScratch := []string{}
			qPusherGamepads := &survey.MultiSelect{
				Message: "Assign gamepads to " + pusherName,
				Options: func() []string {
					qs := []string{}
					for q := range quadScratch {
						qs = append(qs, q)
					}
					sort.Strings(qs)
					return qs
				}(),
			}
			if err := survey.AskOne(qPusherGamepads, &pusherScratch); err != nil {
				fmt.Println(err.Error)
				return
			}

			for _, q := range pusherScratch {
				cfg.GamepadPushers[q] = pusherName
				delete(quadScratch, q)
			}
		}
	}

	fmt.Println("Your event is configured as follows")
	fmt.Println("")
	fmt.Printf("You have %d field(s)\n", cfg.FieldCount)
	if !cfg.AllGamepadsLocal {
		fmt.Println("You are making use of remote gamepads")
		fmt.Println("The following gamepads are directly attached system")
		lScratch := []string{}
		for g, p := range cfg.GamepadPushers {
			if p == "self" {
				lScratch = append(lScratch, g)
			}
		}

		sort.Strings(lScratch)
		for _, g := range lScratch {
			fmt.Printf("\t%s\n", g)
		}

		rpusherTmp := make(map[string][]string)
		for q, p := range cfg.GamepadPushers {
			rpusherTmp[p] = append(rpusherTmp[p], q)
		}
		for pusher, quads := range rpusherTmp {
			if pusher == "self" {
				continue
			}
			fmt.Println()
			fmt.Printf("The following gamepads are attached to remote pusher '%s'\n", pusher)
			sort.Strings(quads)
			for _, quad := range quads {
				fmt.Printf("\t%s\n", quad)
			}
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
	gIndex := make(map[string]int)
	for id, quadName := range cfg.Quads {
		pusher := cfg.GamepadPushers[quadName]
		quads[id] = quad{Name: quadName, Gamepad: gIndex[pusher], Pusher: pusher}
		gIndex[pusher]++
	}
	viper.Set("quadrants", quads)
	viper.SetConfigType("yaml")
	viper.WriteConfigAs("config.yml")
}
