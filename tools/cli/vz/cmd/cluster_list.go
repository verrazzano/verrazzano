// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	cluster_client "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	pkg2 "github.com/verrazzano/verrazzano/tools/cli/vz/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const vmcNamespace = "verrazzano-mc"

func init() {
	clusterCmd.AddCommand(clusterListCmd)
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the clusters",
	Long:  "List the clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := listClusters(args); err != nil {
			return err
		}
		return nil
	},
}

func listClusters(args []string) error {

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
		fmt.Println(pkg2.NothingFound)
		return nil
	}

	// print out details of the clusters
	headings := []string{"NAME", "AGE", "APISERVER"}
	data := [][]string{}
	for _, vmc := range vmcs.Items {
		rowData := []string{
			vmc.Name,
			pkg2.Age(vmc.CreationTimestamp),
			vmc.Status.APIUrl,
		}
		data = append(data, rowData)
	}

	// print out the data
	pkg2.PrintTable(headings, data)
	return nil
}
