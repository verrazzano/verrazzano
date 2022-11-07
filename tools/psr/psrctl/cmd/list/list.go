// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
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

var scenarioID string
var namespace string

func NewCmdList(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdList(cmd, vzHelper)
	}
	cmd.Args = cobra.ExactArgs(0)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().StringVarP(&namespace, constants.FlagNamespace, constants.FlagNamespaceShort, "", constants.FlagNamespaceHelp)

	return cmd
}

// RunCmdList - Run the "psrctl List" command
func RunCmdList(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	m, err := scenario.NewManager(namespace)

	if err != nil {
		return fmt.Errorf("Failed to create scenario Manager %v", err)
	}

	scenarios, err := m.FindRunningScenarios()
	if err != nil {
		return fmt.Errorf("Failed to find running scenarios %s: %v", scenarioID, err)
	}
	if len(scenarios) == 0 {
		if len(namespace) == 0 {
			fmt.Println("There are no scenarios running in the cluster")
		} else {
			fmt.Printf("There are no scenarios running in namespace %s\n", namespace)
		}
		return nil
	}

	fmt.Println()
	fmt.Println("Scenarios running in the cluster...")
	for _, sc := range scenarios {
		fmt.Println("----------------")
		fmt.Printf("%s %s\n", "ID: ", sc.ID)
		fmt.Printf("%s %s\n", "Description: ", sc.Description)
		fmt.Println("Helm releases...")
		for _, h := range sc.HelmReleases {
			fmt.Printf("%s/%s\n", h.Namespace, h.Name)
		}
		fmt.Println()
	}

	return nil
}
