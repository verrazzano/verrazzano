// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package run

import (
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "run"
	helpShort   = "Run a PSR test scenario"
	helpLong    = `The command 'run' executes a PSR test scenario consisting of one or more use cases`
	helpExample = `
psrctl start scenario-1`
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
	return nil
}
