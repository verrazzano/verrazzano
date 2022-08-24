// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/capi"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	clusterSubCommandName = "cluster"
	clusterHelpShort      = "Verrazzano create cluster"
	clusterHelpLong       = `The command 'create cluster' provisions a new local Kind cluster with the specified name (defaults to ` + constants.ClusterNameFlagDefault + `)`
	clusterHelpExample    = `vz create cluster [ --name mycluster ]`
)

func newSubcmdCluster(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, clusterSubCommandName, clusterHelpShort, clusterHelpLong)
	cmd.Example = clusterHelpExample
	cmd.PersistentFlags().String(constants.ClusterNameFlagName, constants.ClusterNameFlagDefault, constants.ClusterNameFlagHelp)
	cmd.PersistentFlags().String(constants.ClusterTypeFlagName, constants.ClusterTypeFlagDefault, constants.ClusterTypeFlagHelp)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdCreateCluster(cmd, args)
	}

	return cmd
}

func runCmdCreateCluster(cmd *cobra.Command, args []string) error {
	clusterName, err := cmd.PersistentFlags().GetString(constants.ClusterNameFlagName)
	clusterType, err := cmd.PersistentFlags().GetString(constants.ClusterTypeFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterNameFlagName, err)
	}
	cluster, err := capi.NewBoostrapCluster(capi.ClusterConfigInfo{
		ClusterName: clusterName,
		Type:        clusterType,
	})
	if err != nil {
		return err
	}
	return cluster.Create()
}
