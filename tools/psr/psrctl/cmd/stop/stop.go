// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stop

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
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

const (
	flagScenario       = "scenarioID"
	flagsScenarioShort = "s"
	flagScenarioHelp   = "specifies the scenario ID"
)

var scenarioID string

func NewCmdStop(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdStop(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, flagScenario, flagsScenarioShort, "", flagScenarioHelp)

	return cmd
}

// RunCmdStop - Run the "psrctl Stop" command
func RunCmdStop(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	m := scenario.Manager{
		Log:       vzlog.DefaultLogger(),
		Manifest:  embedded.PsrManifests{},
		Namespace: "default",
	}

	scman, err := m.FindScenarioManifestByID(scenarioID)
	if err != nil {
		return fmt.Errorf("Failed to find scenario %s: %v", scenarioID, err)
	}
	if scman == nil {
		return fmt.Errorf("Failed to find scenario with ID %s", scenarioID)
	}

	fmt.Printf("Starting scenario %s\n", scman.ID)
	msg, err := m.InstallScenario(scman)
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to Stop scenario %s: %v\n%s", scenarioID, err, msg)
	}
	fmt.Printf("ScenarioManifest %s successfully started\n", scman.ID)

	return nil
}
