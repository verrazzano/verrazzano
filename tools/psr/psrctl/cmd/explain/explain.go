// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package explain

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "explain"
	helpShort   = "Explain a PSR test scenario"
	helpLong    = `The command 'explain' explains a available scenario that can be started`
	helpExample = `
psrctl explain scenario-1`
)

func NewCmdExplain(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return explainCmdExplain(cmd, vzHelper)
	}
	cmd.Example = helpExample

	return cmd
}

// explainCmdExplain - explain the "psrctl explain" command
func explainCmdExplain(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println("Listing available scenarios ...")

	scs, err := scenario.ListAvailableScenarios(embedded.Manifests.ScenarioAbsDir)
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}
	for _, sc := range scs {
		fmt.Println()
		fmt.Printf("Name: %s\n", sc.Name)
		fmt.Printf("ID: %s\n", sc.ID)
		fmt.Printf("Description: %s\n", sc.Description)
	}

	return nil
}
