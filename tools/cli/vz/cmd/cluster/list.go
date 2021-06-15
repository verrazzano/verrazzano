// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	cluster_client "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const vmcNamespace = "verrazzano-mc"

type ClusterListOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewClusterListOptions(streams genericclioptions.IOStreams) *ClusterListOptions {
	return &ClusterListOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterList(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClusterListOptions(streams)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the clusters",
		Long:  "List the clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listClusters(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func listClusters(cmd *cobra.Command,args []string) error {

	clientset, err := cluster_client.NewForConfig(pkg.GetKubeConfig())
	if err != nil {
		return err
	}

	vmcs, err := clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	// check if the list is empty
	if len(vmcs.Items) == 0 {
		fmt.Println(helpers.NothingFound)
		return nil
	}

	// print out details of the clusters
	headings := []string{"NAME", "AGE", "DESCRIPTION", "APISERVER"}
	data := [][]string{}
	for _, vmc := range vmcs.Items {
		rowData := []string{
			vmc.Name,
			helpers.Age(vmc.CreationTimestamp),
			vmc.Spec.Description,
			vmc.Status.APIUrl,
		}
		data = append(data, rowData)
	}

	// print out the data
	if err := helpers.PrintTable(headings, data, cmd); err != nil {
		return err
	}
	return nil
}
