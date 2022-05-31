// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"fmt"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "uninstall"
	helpShort   = "Uninstall Verrazzano"
	helpLong    = `Uninstall the Verrazzano Platform Operator and all of the currently installed components.

For example:

TBD
`
)

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Run = runCmdVersion

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
