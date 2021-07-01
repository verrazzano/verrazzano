// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewClusterOptions(streams genericclioptions.IOStreams) *ClusterOptions {
	return &ClusterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdCluster(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Information about clusters",
		Long:  "Information about clusters",
	}

	cmd.AddCommand(NewCmdClusterList(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdClusterGet(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdClusterRegister(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdClusterDeregister(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdClusterManifest(streams, kubernetesInterface))
	return cmd
}
