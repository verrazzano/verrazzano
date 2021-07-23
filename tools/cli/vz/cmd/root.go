// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	verrazzanoclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/app"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/cluster"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/project"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type RootOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func (c *RootOptions) GetKubeConfig() (*rest.Config,error) {
	return helpers.GetKubeConfig()
}

func (c *RootOptions) NewClustersClientSet() (clustersclientset.Interface, error) {
	var client clustersclientset.Interface
	kubeConfig,err := c.GetKubeConfig()
	if err!= nil {
		return client,err
	}
	client, err = clustersclientset.NewForConfig(kubeConfig)
	return client, err
}

func (c *RootOptions) NewProjectClientSet() (projectclientset.Interface, error) {
	var client projectclientset.Interface
	kubeConfig,err := c.GetKubeConfig()
	if err!=nil {
		return client,err
	}
	client, err = projectclientset.NewForConfig(kubeConfig)
	return client, err
}

func (c *RootOptions) NewVerrazzanoClientSet() (verrazzanoclientset.Interface, error) {
	var client verrazzanoclientset.Interface
	kubeConfig,err := c.GetKubeConfig()
	if err!=nil {
		return client,err
	}
	client, err = verrazzanoclientset.NewForConfig(kubeConfig)
	return client, err
}

func (c *RootOptions) NewClientSet() (kubernetes.Interface,error) {
	return helpers.GetKubernetesClientset()
}

func NewRootOptions(streams genericclioptions.IOStreams) *RootOptions {
	return &RootOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdRoot(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewRootOptions(streams)
	cmd := &cobra.Command{
		Use:   "vz",
		Short: "Verrazzano CLI",
		Long:  "Verrazzano CLI",
	}

	cmd.AddCommand(project.NewCmdProject(streams, o))
	cmd.AddCommand(cluster.NewCmdCluster(streams, o))
	cmd.AddCommand(app.NewCmdApp(streams))
	return cmd
}
