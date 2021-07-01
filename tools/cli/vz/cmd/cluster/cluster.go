// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/spf13/cobra"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClusterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func (c *ClusterOptions) GetKubeConfig() *rest.Config {
	return pkg.GetKubeConfig()
}

func (c *ClusterOptions) NewClustersClientSet() (clientset.Interface, error) {
	client, err := clientset.NewForConfig(c.GetKubeConfig())
	return client, err
}

func (c *ClusterOptions) NewClientSet() kubernetes.Interface {
	return pkg.GetKubernetesClientset()
}

func NewClusterOptions(streams genericclioptions.IOStreams) *ClusterOptions {
	return &ClusterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClusterOptions(streams)
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Information about clusters",
		Long:  "Information about clusters",
	}

	cmd.AddCommand(NewCmdClusterList(streams, o))
	cmd.AddCommand(NewCmdClusterGet(streams, o))
	cmd.AddCommand(NewCmdClusterRegister(streams, o))
	cmd.AddCommand(NewCmdClusterDeregister(streams, o))
	cmd.AddCommand(NewCmdClusterManifest(streams, o))
	return cmd
}
