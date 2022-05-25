// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"

	helpers2 "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "version"
	helpShort   = "Verrazzano version information"
	helpLong    = `Command 'version' reports information about the version of the VZ image being run.

For example:

vz version
`
)

func NewCmdVersion(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := helpers2.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Run = runCmdVersion

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
