// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/spf13/cobra"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	restclient "k8s.io/client-go/rest"
)

type ClusterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func (o *ClusterOptions) GetKubeConfig() *restclient.Config{
	return pkg.GetKubeConfig()
}

func (o *ClusterOptions) NewClientSet() (clientset.Interface, error) {
	client, err := clientset.NewForConfig(o.GetKubeConfig())
	return client, err
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
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(NewCmdClusterList(streams))
	cmd.AddCommand(NewCmdClusterRegister(streams, o))
	//cmd.AddCommand(NewCmdClusterDeregister(streams))
	//cmd.AddCommand(NewCmdClusterManifest(streams))
	return cmd
}
