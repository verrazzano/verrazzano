// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var gitCommit = "abcde1234"
var buildDate = "YYYY-MM-DD"
var cliVersion = "vMajor.Minor.Patch"

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
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "Version: v%s\n", cliVersion)
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "BuildDate: %s\n", buildDate)
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "GitCommit: %s\n", gitCommit)
	return nil
}
