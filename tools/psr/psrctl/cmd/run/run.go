// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package run

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
	CommandName = "run"
	helpShort   = "Run a PSR test scenario"
	helpLong    = `The command 'run' executes a PSR test scenario consisting of one or more use cases`
	helpExample = `psrctl run -s ops-s1`
)

const (
	flagScenario       = "scenarioID"
	flagsScenarioShort = "s"
	flagScenarioHelp   = "specifies the scenario ID"
)

var scenarioID string

func NewCmdRun(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdRun(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, flagScenario, flagsScenarioShort, "", flagScenarioHelp)

	return cmd
}

// runCmdRun - run the "psrctl run" command
func runCmdRun(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	sc, err := scenario.FindScenarioByID(embedded.Manifests.ScenarioAbsDir, scenarioID)
	if err != nil {
		return fmt.Errorf("Failed to find scenario %s: %v", scenarioID, err)
	}
	if sc == nil {
		return fmt.Errorf("Failed to find scenario with ID %s", scenarioID)
	}

	fmt.Printf("Starting scenario %s\n", sc.ID)
	msg, err := scenario.InstallScenario(vzlog.DefaultLogger(), embedded.Manifests, sc, "default")
	if err != nil {
		// Cobra will display failure message
		return fmt.Errorf("Failed to run scenario %s: %v\n%s", scenarioID, err, msg)
	}
	fmt.Printf("ScenarioManifest %s successfully started\n", sc.ID)

	return nil
}
