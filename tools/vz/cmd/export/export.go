// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package export

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/export/oam"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "export"
	helpShort   = "Export content"
	helpLong    = `Export the yaml of the subcomponent specified.`
	helpExample = `
TBD
`
)

func NewCmdExport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Example = helpExample

	// Add commands
	cmd.AddCommand(oam.NewCmdExportOAM(vzHelper))

	return cmd
}
