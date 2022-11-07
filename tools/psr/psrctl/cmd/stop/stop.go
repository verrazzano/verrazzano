// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stop

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "stop"
	helpShort   = "Stop a PSR scenario"
	helpLong    = `The command 'stop' stops a PSR scenario`
	helpExample = `psrctl stop -s ops-s1`
)

var scenarioID string
var namespace string

func NewCmdStop(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdStop(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().StringVarP(&namespace, constants.FlagNamespace, constants.FlagNamespaceShort, "default", constants.FlagNamespaceHelp)

	return cmd
}

// RunCmdStop - Run the "psrctl Stop" command
func RunCmdStop(cmd *cobra.Command, vzHelper helpers.VZHelper) error {

	m, err := scenario.NewManager(namespace)
	if err != nil {
		return fmt.Errorf("Failed to create scenario Manager %v", err)
	}

	fmt.Printf("Stopping scenario %s\n", scenarioID)
	msg, err := m.StopScenarioByID(scenarioID)
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to stop scenario %s/%s: %v\n%s", namespace, scenarioID, err, msg)
	}
	fmt.Printf("Scenario %s successfully stopped\n", scenarioID)

	return nil
}
