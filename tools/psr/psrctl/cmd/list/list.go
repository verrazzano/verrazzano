// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

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
	CommandName = "list"
	helpShort   = "List all of the running PSR scenarios"
	helpLong    = `The command 'list' lists the PSR scenarios that are running in the cluster`
	helpExample = `psrctl list `
)

const (
	flagScenario       = "scenarioID"
	flagsScenarioShort = "s"
	flagScenarioHelp   = "specifies the scenario ID"
)

var scenarioID string

func NewCmdList(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdList(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, flagScenario, flagsScenarioShort, "", flagScenarioHelp)

	return cmd
}

// RunCmdList - Run the "psrctl List" command
func RunCmdList(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	m := scenario.Manager{
		Log:       vzlog.DefaultLogger(),
		Manifest:  embedded.PsrManifests{},
		Namespace: "",
	}

	scenarios, err := m.FindRunningScenarios()
	if err != nil {
		return fmt.Errorf("Failed to find running scenarios %s: %v", scenarioID, err)
	}
	if len(scenarios) == 0 {
		fmt.Println("There are no scenarios running in the cluster")
		return nil
	}

	fmt.Println("Running Scenarios...")
	for _, sc := range scenarios {
		fmt.Println(sc.ID, sc.Description)
	}

	return nil
}
