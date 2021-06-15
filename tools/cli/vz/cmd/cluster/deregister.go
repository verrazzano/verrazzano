// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	cluster_client "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterDeregisterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewClusterDeregisterOptions (streams genericclioptions.IOStreams) *ClusterDeregisterOptions {
	return &ClusterDeregisterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterDeregister(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClusterDeregisterOptions(streams)
	cmd := &cobra.Command{
		Use:   "deregister [name]",
		Short: "Deregister a managed cluster",
		Long:  "Deregister a managed cluster",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deregisterCluster(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func deregisterCluster(cmd *cobra.Command, args []string) error {

	//name of vmc resource
	vmcName := args[0]

	//get the vmc resource and delete it
	config := pkg.GetKubeConfig()
	clientset, err := cluster_client.NewForConfig(config)

	if err != nil {
		return err
	}

	err = clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).Delete(context.Background(), vmcName, v1.DeleteOptions{})

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), vmcName + " deregistered")
	return err
}
