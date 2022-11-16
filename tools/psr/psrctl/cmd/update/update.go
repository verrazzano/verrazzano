// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"
	"github.com/spf13/cobra"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "update"
	helpShort   = "Update a running PSR scenario configuration"
	helpLong    = `The command 'update' updates the configuration of a running PSR scenario.  
The underlying use case helm charts will be updated using the "helm upgrade --reuse-values" command.  
If you provide any overrides then they will be applied to all the helm charts in the scenario.  
The only way to modify a use case specific configuration is to put the changes in the scenario files 
and apply them.  For example, if you are running a scenario with the -d parameter providing 
a custom scenario, you can modify those scenario files and update the running scenario.  
You cannot change the scenario.yaml file, you can only change the usecase-override files`
	helpExample = `
// Update the backend image for all workers in running scenario ops-s1
psrctl update -s ops-s1 -w=ghcr.io/verrazzano/psr-backend:xyz
`
)

var scenarioID string
var namespace string
var workerImage string
var imagePullSecret string

func NewCmdUpdate(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdUpdate(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().StringVarP(&namespace, constants.FlagNamespace, constants.FlagNamespaceShort, "default", constants.FlagNamespaceHelp)
	cmd.PersistentFlags().StringVarP(&workerImage, constants.WorkerImageName, constants.WorkerImageNameShort, constants.GetDefaultWorkerImage(), constants.WorkerImageNameHelp)
	cmd.PersistentFlags().StringVarP(&imagePullSecret, constants.ImagePullSecretName, constants.ImagePullSecretNameShort, constants.ImagePullSecDefault, constants.ImagePullSecretNameHelp)

	return cmd
}

// RunCmdUpdate - update the "psrctl update" command
func RunCmdUpdate(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	m, err := scenario.NewManager(namespace, "", buildHelmOverrides()...)
	if err != nil {
		return fmt.Errorf("Failed to create scenario Manager %v", err)
	}

	fmt.Printf("Updating scenario %s\n", scenarioID)
	msg, err := m.UpdateScenario(scenarioID)
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to update scenario %s/%s: %v\n%s", namespace, scenarioID, err, msg)
	}
	fmt.Printf("Scenario %s successfully updated\n", scenarioID)

	return nil
}

func buildHelmOverrides() []helmcli.HelmOverrides {
	return []helmcli.HelmOverrides{
		{SetOverrides: fmt.Sprintf("%s=%s", constants.ImageNameKey, workerImage)},
		{SetOverrides: fmt.Sprintf("%s=%s", constants.ImagePullSecKey, imagePullSecret)},
	}
}
