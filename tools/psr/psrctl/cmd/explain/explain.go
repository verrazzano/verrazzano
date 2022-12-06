// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package explain

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "explain"
	helpShort   = "Describe PSR scenarios that can be started"
	helpLong    = `The command 'explain' describes scenarios that can be started.  The scenarios are represented by
manifest files built into the psrctl binary.`
	helpExample = `
psrctl explain 
psrctl explain -s ops-s1`
)

var scenarioID string
var verbose bool

func NewCmdExplain(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdExplain(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().BoolVarP(&verbose, constants.FlagVerbose, constants.FlagVerboseShort, false, constants.FlagVerboseHelp)

	return cmd
}

// RunCmdExplain - explain the "psrctl explain" command
func RunCmdExplain(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println()
	fmt.Println("Listing available scenarios ...")

	m := manifest.ManifestManager{
		Log:      vzlog.DefaultLogger(),
		Manifest: *manifest.Manifests,
	}

	scs, err := m.ListScenarioManifests()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	for _, sc := range scs {
		if len(scenarioID) > 0 && sc.ID != scenarioID {
			continue
		}
		fmt.Println("----------------")
		fmt.Printf("Name: %s\n", sc.Name)
		fmt.Printf("ID: %s\n", sc.ID)
		fmt.Printf("Description: %s\n", sc.Description)

		// If verbose
		if verbose {
			fmt.Println("Use cases:")
			for _, uc := range sc.Usecases {
				fmt.Printf("Usecase path %s:  Description: %s\n", uc.UsecasePath, uc.Description)
			}
		}
		if len(scenarioID) > 0 && sc.ID == scenarioID {
			break
		}
	}
	fmt.Println()

	return nil
}
