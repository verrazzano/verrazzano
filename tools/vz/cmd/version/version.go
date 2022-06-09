// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var gitCommit = "MOOSE"

const (
	CommandName = "version"
	helpShort   = "Verrazzano version information"
	helpLong    = `The command 'version' reports information about the version of the vz tool being run`
	helpExample = `vz version`
)

func NewCmdVersion(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Example = helpExample
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdVersion(cmd, args, vzHelper)
	}

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), "Not implemented yet\n")
	fmt.Fprintf(vzHelper.GetOutputStream(), "testing %s value \n", gitCommit)
	return nil
}
