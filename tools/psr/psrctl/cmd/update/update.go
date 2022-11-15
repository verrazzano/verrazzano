// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "update"
	helpShort   = "Update the configuration of a running PSR scenario"
	helpLong    = `The command 'update' updates the configuration of a running PSR scenario.  
The underlying use case helm charts will be updated using the "helm upgrade --reuse-values" command.  
If you provide any overrides then they will be applied to all the helm charts in the scenario.  
The only way to modify a use case specific configuration is to put the changes in the scenario files 
and apply them.  For example, if you are running a scenario with the -d parameter providing 
a custom scenario, you can modify those scenario files and update the running scenario.  
You cannot change the scenario.yaml file, you can only change the usecase-override files`
	helpExample = `
// Update the backend image for all running scenarios
psrctl update -s ops-s1 --set imageName=ghcr.io/verrazzano/psr-backend:xyz

// Update the scenario usecase overrides for a custom scenario
psrctl update -s custom-s1 -d mycustom-scenario-dir`
)

var scenarioID string
var verbose bool

func NewCmdUpdate(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return updateCmdUpdate(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().BoolVarP(&verbose, constants.FlagVerbose, constants.FlagVerboseShort, true, constants.FlagVerboseHelp)

	return cmd
}

// updateCmdUpdate - update the "psrctl update" command
func updateCmdUpdate(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println()
	fmt.Println("Listing available scenarios ...")

	m := scenario.Manager{
		Namespace: "default",
		Log:       vzlog.DefaultLogger(),
		Manifest:  *embedded.Manifests,
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
				fmt.Printf("Usecase path %s :  Description: %s\n", uc.UsecasePath, uc.Description)
			}
		}
		if len(scenarioID) > 0 && sc.ID == scenarioID {
			break
		}
	}
	fmt.Println()

	return nil
}
