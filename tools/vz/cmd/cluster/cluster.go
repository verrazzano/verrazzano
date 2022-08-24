// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var cliVersion string
var buildDate string
var gitCommit string

const (
	CommandName = "cluster"
	HelpShort   = "Verrazzano cluster operations"
	helpLong    = `The command 'cluster <subcommand>' performs the cluster operation specified by the subcommand`
	helpExample = `vz cluster <subcommand>`
	hidden      = true
)

func NewCmdCluster(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, HelpShort, helpLong)
	cmd.Hidden = hidden
	addSubCommandsCluster(vzHelper, cmd)
	cmd.Example = helpExample
	return cmd
}

func addSubCommandsCluster(vzHelper helpers.VZHelper, parentCmd *cobra.Command) {
	parentCmd.AddCommand(newSubcmdCreate(vzHelper))
}
