// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package run

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "run"
	helpShort   = "Run a PSR test scenario"
	helpLong    = `The command 'run' executes a PSR test scenario consisting of one or more use cases`
	helpExample = `
psrctl run scenario-1`
)

const (
	flagScenario       = "scenario"
	flagsScenarioShort = "s"
	flagScenarioHelp   = "specifies the scenario ID"
)

var scenarioName string

func NewCmdRun(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdRun(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioName, flagScenario, flagsScenarioShort, "", flagScenarioHelp)

	return cmd
}

// runCmdRun - run the "psrctl run" command
func runCmdRun(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println("Runing example scenario...")

	sc, err := scenario.FindScenarioByName(embedded.Manifests.ScenarioAbsDir, scenarioName)
	if err != nil {
		fmt.Printf("Failed to find scenario %s: %v", scenarioName, err)
		return err
	}

	_, err = scenario.InstallScenario(embedded.Manifests, sc)
	if err != nil {
		fmt.Printf("Failed to find scenario %s: %v", scenarioName, err)
		return err
	}

	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	fmt.Println("Example Scenario successfully installed")

	return nil
}
