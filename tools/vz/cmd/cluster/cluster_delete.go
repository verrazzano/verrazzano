// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/capi"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	deleteSubCommandName = "delete"
	deleteHelpShort      = "Verrazzano cluster delete"
	deleteHelpLong       = `The command 'cluster delete' destroys the local cluster with the given name (defaults to "` + constants.ClusterNameFlagDefault + `")`
	deleteHelpExample    = `vz cluster delete --name mycluster`
)

func newSubcmdDelete(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, deleteSubCommandName, deleteHelpShort, deleteHelpLong)
	cmd.Example = deleteHelpExample
	cmd.PersistentFlags().String(constants.ClusterNameFlagName, constants.ClusterNameFlagDefault, constants.ClusterNameFlagHelp)

	// add a hidden cluster type flag for testing purposes, even though delete does not require it, with an empty default so that
	// the underlying CAPI default is used if unspecified
	cmd.PersistentFlags().String(constants.ClusterTypeFlagName, "", constants.ClusterTypeFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.ClusterTypeFlagName)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdClusterDelete(cmd, args)
	}

	return cmd
}

func runCmdClusterDelete(cmd *cobra.Command, args []string) error {
	clusterName, err := cmd.PersistentFlags().GetString(constants.ClusterNameFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterNameFlagName, err)
	}

	clusterType, err := cmd.PersistentFlags().GetString(constants.ClusterTypeFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterTypeFlagName, err)
	}

	cluster, err := capi.NewBoostrapCluster(capi.ClusterConfig{
		ClusterName: clusterName,
		Type:        clusterType,
	})
	if err != nil {
		return err
	}
	return cluster.Destroy()
}
