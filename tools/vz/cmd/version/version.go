// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const CommandName = "version"

func NewCmdVersion(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := helpers.NewCommand(vzHelper, CommandName, "Verrazzano version information", "Verrazzano version information")
	cmd.Run = runCmdVersion

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
