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
	createSubCommandName = "create"
	createHelpShort      = "Verrazzano cluster create"
	createHelpLong       = `The command 'cluster create' provisions a new local cluster with the given name and type (defaults to "` + constants.ClusterNameFlagDefault + `" and "` + capi.KindClusterType + `")`
	createHelpExample    = `vz cluster create --name mycluster --type ` + capi.KindClusterType
)

func newSubcmdCreate(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, createSubCommandName, createHelpShort, createHelpLong)
	cmd.Example = createHelpExample
	cmd.PersistentFlags().String(constants.ClusterNameFlagName, constants.ClusterNameFlagDefault, constants.ClusterNameFlagHelp)
	cmd.PersistentFlags().String(constants.ClusterTypeFlagName, constants.ClusterTypeFlagDefault, constants.ClusterTypeFlagHelp)
	cmd.PersistentFlags().String(constants.ClusterImageFlagName, constants.ClusterImageFlagDefault, constants.ClusterImageFlagHelp)
	// the image flag should be hidden since it is not intended for general use
	cmd.PersistentFlags().MarkHidden(constants.ClusterImageFlagName)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdClusterCreate(cmd, args)
	}

	return cmd
}

func runCmdClusterCreate(cmd *cobra.Command, args []string) error {
	clusterName, err := cmd.PersistentFlags().GetString(constants.ClusterNameFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterNameFlagName, err)
	}

	clusterType, err := cmd.PersistentFlags().GetString(constants.ClusterTypeFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterTypeFlagName, err)
	}

	clusterImg, err := cmd.PersistentFlags().GetString(constants.ClusterImageFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterImageFlagName, err)
	}

	cluster, err := capi.NewBoostrapCluster(capi.ClusterConfigInfo{
		ClusterName:    clusterName,
		Type:           clusterType,
		ContainerImage: clusterImg,
	})
	if err != nil {
		return err
	}
	if err := cluster.Create(); err != nil {
		return err
	}
	fmt.Printf("Cluster %s created successfully, initializing...\n", clusterName)
	if err := cluster.Init(); err != nil {
		return err
	}
	fmt.Println("Cluster initialization complete")
	return nil
}
