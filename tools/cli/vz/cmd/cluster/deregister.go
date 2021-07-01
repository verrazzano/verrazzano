// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterDeregisterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewClusterDeregisterOptions(streams genericclioptions.IOStreams) *ClusterDeregisterOptions {
	return &ClusterDeregisterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterDeregister(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewClusterDeregisterOptions(streams)
	cmd := &cobra.Command{
		Use:   "deregister [name]",
		Short: "Deregister a managed cluster",
		Long:  "Deregister a managed cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.deregisterCluster(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func (o *ClusterDeregisterOptions) deregisterCluster(kubernetesInterface helpers.Kubernetes) error {

	// Name of vmc resource
	vmcName := o.args[0]

	// Get the vmc resource and delete it
	clientset, err := kubernetesInterface.NewClustersClientSet()

	if err != nil {
		return err
	}

	err = clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).Delete(context.Background(), vmcName, v1.DeleteOptions{})

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(o.Out, vmcName+" deregistered")
	return err
}
