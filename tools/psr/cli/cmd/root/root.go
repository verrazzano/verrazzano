// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/cli/cmd/run"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var kubeconfig string
var context string

const (
	CommandName = "psrctl"
	helpShort   = "The psrctl tool is a command-line utility that allows Verrazzano operators to run PSR tests against a Verrazzano environment"
	helpLong    = "The psrctl tool is a command-line utility that allows Verrazzano operators to run PSR tests against a Verrazzano environment"
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	// Add global flags
	cmd.PersistentFlags().StringVar(&kubeconfig, constants.GlobalFlagKubeConfig, "", constants.GlobalFlagKubeConfigHelp)
	cmd.PersistentFlags().StringVar(&context, constants.GlobalFlagContext, "", constants.GlobalFlagContextHelp)

	// Add commands
	cmd.AddCommand(run.NewCmdRun(vzHelper))

	return cmd
}
