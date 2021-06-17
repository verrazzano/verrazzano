// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const vmcNamespace = "verrazzano-mc"

type ClusterListOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
	outputOptions string
}

func NewClusterListOptions(streams genericclioptions.IOStreams) *ClusterListOptions {
	return &ClusterListOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterList(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewClusterListOptions(streams)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the clusters",
		Long:  "List the clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.listClusters(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	//o.configFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.outputOptions, "output", "o", "", helpers.OutputUsage)
	return cmd
}

func (o *ClusterListOptions) listClusters(kubernetesInterface helpers.Kubernetes) error {

	clientset, err := kubernetesInterface.NewClientSet()
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

	//Output options was specified
	if len(o.outputOptions) != 0 {
		//data, err := vmcs.Marshal()
		err = helpers.PrintJsonYaml(o.outputOptions, vmcs, o.Out)
		return err
	}

	// print out details of the clusters
	headings := []string{"NAME", "AGE", "STATUS", "DESCRIPTION", "APISERVER"}
	data := [][]string{}
	for _, vmc := range vmcs.Items {
		rowData := []string{
			vmc.Name,
			helpers.Age(vmc.CreationTimestamp),
			getStatus(vmc.Status.Conditions[0].Status),
			vmc.Spec.Description,
			vmc.Status.APIUrl,
		}
		data = append(data, rowData)
	}

	// print out the data
	if err := helpers.PrintTable(headings, data, o.Out); err != nil {
		return err
	}
	return nil
}

func getStatus(status corev1.ConditionStatus) string {
	switch status {
	case "True":
		return "Ready"
	case "False":
		return "Not Ready"
	case "Unknown":
		return ""
	default:
		panic("shouldn't reach here")
	}
}

