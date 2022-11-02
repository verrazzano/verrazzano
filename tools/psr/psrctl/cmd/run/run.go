// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package run

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/helm"
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

func NewCmdRun(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdRun(cmd, vzHelper)
	}
	cmd.Example = helpExample

	return cmd
}

// runCmdRun - run the "psrctl run" command
func runCmdRun(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Println("Running example scenario...")

	msg, err := helm.InstallScenario()
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Println("Helm results...")
	fmt.Println()
	fmt.Println(msg)
	fmt.Println()
	fmt.Println("Example Scenario successfully installed")

	return nil
}
