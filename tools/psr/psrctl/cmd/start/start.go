// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"fmt"

	"github.com/spf13/cobra"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "start"
	helpShort   = "Start a PSR scenario"
	helpLong    = `The command 'start' starts a PSR scenario in the specified namespace. 
Multiple scenarios can be started in the same namespace.`
	helpExample = `psrctl start -s ops-s1`
)

var scenarioID string
var namespace string
var scenarioDir string
var workerImage string
var imagePullSecret string

func NewCmdStart(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdStart(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().StringVarP(&namespace, constants.FlagNamespace, constants.FlagNamespaceShort, "default", constants.FlagNamespaceHelp)
	cmd.PersistentFlags().StringVarP(&scenarioDir, constants.FlagScenarioDir, constants.FlagScenarioDirShort, "", constants.FlagScenarioDirHelp)
	cmd.PersistentFlags().StringVarP(&workerImage, constants.WorkerImageName, constants.WorkerImageNameShort, constants.GetDefaultWorkerImage(), constants.WorkerImageNameHelp)
	cmd.PersistentFlags().StringVarP(&imagePullSecret, constants.ImagePullSecretName, constants.ImagePullSecretNameShort, constants.ImagePullSecDefault, constants.ImagePullSecretNameHelp)

	return cmd
}

// RunCmdStart - Run the "psrctl start" command
func RunCmdStart(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	// GetscenarioManifestifest gets the scenarioManifestifest for the given scenarioID
	manifestManager, err := manifest.NewManager(scenarioDir)
	if err != nil {
		return fmt.Errorf("Failed to create scenario scenarioManifestanger %v", err)
	}
	scenarioManifest, err := manifestManager.FindScenarioManifestByID(scenarioID)
	if err != nil {
		return fmt.Errorf("Failed to find scenario manifest %s: %v", scenarioID, err)
	}
	if scenarioManifest == nil {
		return fmt.Errorf("Failed to find scenario manifest with ID %s", scenarioID)
	}

	scenarioManager, err := scenario.NewManager(namespace, buildHelmOverrides()...)
	if err != nil {
		return fmt.Errorf("Failed to create scenario scenarioManifestanger %v", err)
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Starting scenario %s\n", scenarioManifest.ID)
	msg, err := scenarioManager.StartScenario(manifestManager, scenarioManifest, vzHelper)
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to start scenario %s/%s: %v\n%s", namespace, scenarioID, err, msg)
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), "Scenario %s successfully started\n", scenarioManifest.ID)

	return nil
}

func buildHelmOverrides() []helmcli.HelmOverrides {
	var ov []helmcli.HelmOverrides
	if len(workerImage) > 0 {
		ov = append(ov, helmcli.HelmOverrides{SetOverrides: fmt.Sprintf("%s=%s", constants.ImageNameKey, workerImage)})
	}
	if len(imagePullSecret) > 0 {
		ov = append(ov, helmcli.HelmOverrides{SetOverrides: fmt.Sprintf("%s=%s", constants.ImagePullSecKey, imagePullSecret)})
	}
	return ov
}
