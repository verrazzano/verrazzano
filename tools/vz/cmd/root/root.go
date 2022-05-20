// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var kubeconfig string
var context string

const (
	CommandName = "vz"
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := helpers.NewCommand(vzHelper, CommandName, "Verrazzano CLI", "Verrazzano CLI")

	// Add global flags
	cmd.PersistentFlags().StringVarP(&kubeconfig, constants.GlobalFlagKubeConfig, "c", "", "Kubernetes configuration file")
	cmd.PersistentFlags().StringVar(&context, constants.GlobalFlagContext, "", "The name of the kubeconfig context to use")

	// Add commands
	cmd.AddCommand(status.NewCmdStatus(vzHelper))
	cmd.AddCommand(version.NewCmdVersion(vzHelper))

	return cmd
}
