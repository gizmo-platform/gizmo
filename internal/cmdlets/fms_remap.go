package cmdlets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var (
	fmsRemapCmd = &cobra.Command{
		Use:   "remap",
		Short: "remap provides a means of immediately remapping teams",
		Long:  fmsRemapCmdLongDocs,
		Run:   fmsRemapCmdRun,
	}

	fmsRemapCmdLongDocs = `remap is used to insert an immediate update to the fms/team mapping
table.  This will disrupt any teams currently on the fms!`
)

func init() {
	fmsCmd.AddCommand(fmsRemapCmd)
}

func fmsRemapCmdRun(c *cobra.Command, args []string) {
	fAddr := os.Getenv("GIZMO_FMS_ADDR")
	if fAddr == "" {
		fAddr = "localhost:8080"
	}

	r, err := http.Get("http://" + fAddr + "/admin/cfg/quads")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting quads: %s\n", err)
		os.Exit(2)
	}

	quads := []string{}
	if err := json.NewDecoder(r.Body).Decode(&quads); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting quads: %s\n", err)
		os.Exit(2)
	}
	r.Body.Close()

	r, err = http.Get("http://" + fAddr + "/admin/map/current")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting map: %s\n", err)
		os.Exit(2)
	}

	cMap := make(map[string]string)
	if err := json.NewDecoder(r.Body).Decode(&cMap); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting map: %s\n", err)
		os.Exit(2)
	}
	ccMap := make(map[string]string, len(cMap))
	r.Body.Close()

	if len(cMap) > 0 {
		fmt.Println("Current Mapping:")
		for team, quad := range cMap {
			fmt.Printf("  %s:\t%s\n", quad, team)
			ccMap[quad] = team
		}
		fmt.Println()
	}
	fmt.Println("Enter new mapping")

	tNumValidator := func(a interface{}) error {
		if a.(string) == "-" {
			return nil
		}
		if _, err := strconv.Atoi(a.(string)); err != nil {

			return errors.New("team number must be a number")
		}
		return nil
	}
	qMap := []*survey.Question{}
	for _, quad := range quads {
		qMap = append(qMap, &survey.Question{
			Name:     quad,
			Validate: tNumValidator,
			Prompt: &survey.Input{
				Message: quad,
				Default: ccMap[quad],
			},
		})
	}

	nMap := make(map[string]interface{})
	if err := survey.Ask(qMap, &nMap); err != nil {
		fmt.Fprintf(os.Stderr, "Error polling for fields: %s\n", err)
		os.Exit(2)
	}

	nnMap := make(map[string]string, len(nMap))
	for f, t := range nMap {
		if t == "-" {
			continue
		}
		nnMap[t.(string)] = f
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(nnMap)
	http.Post("http://"+fAddr+"/admin/map/immediate", "application/json", buf)
}
