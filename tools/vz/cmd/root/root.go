// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/analyze"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/cluster"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/uninstall"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/upgrade"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var kubeconfig string
var context string

const (
	CommandName = "vz"
	helpShort   = "The vz tool is a command-line utility that allows Verrazzano operators to query and manage a Verrazzano environment"
	helpLong    = "The vz tool is a command-line utility that allows Verrazzano operators to query and manage a Verrazzano environment"
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	// Add global flags
	cmd.PersistentFlags().StringVar(&kubeconfig, constants.GlobalFlagKubeConfig, "", constants.GlobalFlagKubeConfigHelp)
	cmd.PersistentFlags().StringVar(&context, constants.GlobalFlagContext, "", constants.GlobalFlagContextHelp)

	// Add commands
	cmd.AddCommand(status.NewCmdStatus(vzHelper))
	cmd.AddCommand(version.NewCmdVersion(vzHelper))
	cmd.AddCommand(install.NewCmdInstall(vzHelper))
	cmd.AddCommand(upgrade.NewCmdUpgrade(vzHelper))
	cmd.AddCommand(uninstall.NewCmdUninstall(vzHelper))
	cmd.AddCommand(analyze.NewCmdAnalyze(vzHelper))
	cmd.AddCommand(bugreport.NewCmdBugReport(vzHelper))
	cmd.AddCommand(cluster.NewCmdCluster(vzHelper))

	return cmd
}
