// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "list"
	helpShort   = "List the running PSR scenarios"
	helpLong    = `The command 'list' lists the PSR scenarios that are running in the cluster or in the specified namespace`
	helpExample = `
psrctl list 
psrctl list -A
psrctl list -n foo
`
)

var scenarioID string
var namespace string
var allNamepaces bool
var outputFormat string

func NewCmdList(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdList(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&scenarioID, constants.FlagScenario, constants.FlagsScenarioShort, "", constants.FlagScenarioHelp)
	cmd.PersistentFlags().StringVarP(&namespace, constants.FlagNamespace, constants.FlagNamespaceShort, "default", constants.FlagNamespaceHelp)
	cmd.PersistentFlags().BoolVarP(&allNamepaces, constants.FlagAll, constants.FlagAllShort, false, constants.FlagAllHelp)
	cmd.PersistentFlags().StringVarP(&outputFormat, constants.OutputFormatName, constants.OutputFormatNameShort, "text", constants.OutputFormatHelp)
	cmd.PersistentFlags().MarkHidden(constants.OutputFormatName)

	return cmd
}

// RunCmdList - Run the "psrctl List" command
func RunCmdList(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	if allNamepaces {
		namespace = ""
	}
	scenarioMan, err := scenario.NewManager(namespace)

	if err != nil {
		return fmt.Errorf("Failed to create scenario ScenarioMananger %v", err)
	}

	scenarios, err := scenarioMan.FindRunningScenarios()
	if err != nil {
		return fmt.Errorf("Failed to find running scenarios %s: %v", scenarioID, err)
	}
	if len(scenarios) == 0 {
		if len(namespace) == 0 {
			fmt.Fprintln(vzHelper.GetOutputStream(), "There are no scenarios running in the cluster")
		} else {
			fmt.Fprintf(vzHelper.GetOutputStream(), "There are no scenarios running in namespace %s\n", namespace)
		}
		return nil
	}

	if outputFormat == "json" {
		jsonOut, err := json.Marshal(scenarios)
		if err != nil {
			return err
		}
		fmt.Fprint(vzHelper.GetOutputStream(), string(jsonOut))
		return nil
	}

	fmt.Println()
	if len(namespace) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Scenarios running in namespace %s\n", namespace)
	} else {
		fmt.Fprintln(vzHelper.GetOutputStream(), "Scenarios running in the cluster")
	}

	for _, sc := range scenarios {
		fmt.Fprintln(vzHelper.GetOutputStream(), "----------------")
		fmt.Fprintf(vzHelper.GetOutputStream(), "Namespace: %s\n", sc.Namespace)
		fmt.Fprintf(vzHelper.GetOutputStream(), "%s %s\n", "ID: ", sc.ID)
		fmt.Fprintf(vzHelper.GetOutputStream(), "%s %s\n", "Description: ", sc.Description)
		fmt.Fprintln(vzHelper.GetOutputStream(), "Helm releases:")
		for _, h := range sc.HelmReleases {
			fmt.Fprintf(vzHelper.GetOutputStream(), "%s/%s\n", h.Namespace, h.Name)
		}
		fmt.Fprintln(vzHelper.GetOutputStream())
	}

	return nil
}
