// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "start"
	helpShort   = "Start a PSR scenario"
	helpLong    = `The command 'start' starts a PSR scenario`
	helpExample = `psrctl start -s ops-s1`
)

const (
	flagScenario       = "scenarioID"
	flagsScenarioShort = "s"
	flagScenarioHelp   = "specifies the scenario ID"
)

var scenarioID string

func NewCmdStart(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdStart(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, flagScenario, flagsScenarioShort, "", flagScenarioHelp)

	return cmd
}

// RunCmdStart - Run the "psrctl start" command
func RunCmdStart(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	m, err := scenario.NewManager("default")
	if err != nil {
		return fmt.Errorf("Failed to create scenario Manager %v", err)
	}

	scman, err := m.FindScenarioManifestByID(scenarioID)
	if err != nil {
		return fmt.Errorf("Failed to find scenario %s: %v", scenarioID, err)
	}
	if scman == nil {
		return fmt.Errorf("Failed to find scenario with ID %s", scenarioID)
	}

	fmt.Printf("Starting scenario %s\n", scman.ID)
	msg, err := m.StartScenario(scman)
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to start scenario %s: %v\n%s", scenarioID, err, msg)
	}
	fmt.Printf("ScenarioManifest %s successfully started\n", scman.ID)

	return nil
}
