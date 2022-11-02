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
	helpShort   = "Start a PSR test scenario"
	helpLong    = `The command 'start' executes a PSR test scenario consisting of one or more use cases`
	helpExample = `
psrctl start scenario-1`
)

func NewCmdStart(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return startCmdStart(cmd, vzHelper)
	}
	cmd.Example = helpExample

	return cmd
}

// startCmdStart - start the "psrctl start" command
func startCmdStart(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println("Starting example scenario...")

	msg, err := scenario.InstallScenario()

	fmt.Println("Helm results...")
	fmt.Println()
	fmt.Println(msg)
	fmt.Println()

	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	fmt.Println("Example Scenario successfully installed")

	return nil
}
