// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var cliVersion string
var buildDate string
var gitCommit string

const (
	CommandName = "create"
	helpShort   = "Verrazzano create specified resources"
	helpLong    = `The command 'create <subcommand>' creates the resource specified by the subcommand`
	helpExample = `vz create <subcommand>`
	hidden      = true
)

func NewCmdCreate(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Hidden = hidden
	addSubCommandsCreate(vzHelper, cmd)
	cmd.Example = helpExample
	return cmd
}

func addSubCommandsCreate(vzHelper helpers.VZHelper, parentCmd *cobra.Command) {
	parentCmd.AddCommand(newSubcmdCluster(vzHelper))
}
