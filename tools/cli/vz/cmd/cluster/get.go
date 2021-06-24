// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterGetOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewClusterGetOptions(streams genericclioptions.IOStreams) *ClusterGetOptions {
	return &ClusterGetOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		PrintFlags:  helpers.NewGetPrintFlags(),
	}
}

func NewCmdClusterGet(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewClusterGetOptions(streams)
	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Get the managed cluster",
		Long:  "Get the managed cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.getCluster(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func (o *ClusterGetOptions) getCluster(kubernetesInterface helpers.Kubernetes) error {

	vmcName := o.args[0]

	clientset, err := kubernetesInterface.NewClientSet()
	if err != nil {
		return err
	}

	vmcObject, err := clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).Get(context.Background(), vmcName, v1.GetOptions{})
	if err != nil {
		return err
	}

	//Output options was specified
	if len(*o.PrintFlags.OutputFormat) != 0 {
		// Set the Version and Kind before passing it as runtime object
		vmcObject.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "clusters.verrazzano.io",
			Version: "v1alpha1",
			Kind:    "VerrazzanoManagedCluster",
		})

		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			return err
		}
		err = printer.PrintObj(vmcObject, o.Out)

		return err
	}

	// print out details of the cluster
	headings := []string{"NAME", "AGE", "STATUS", "DESCRIPTION", "APISERVER"}
	data := [][]string{
		{
			vmcName,
			helpers.Age(vmcObject.CreationTimestamp),
			getReadyStatus(vmcObject.Status),
			vmcObject.Spec.Description,
			vmcObject.Status.APIUrl,
		},
	}

	// print out the data
	if err := helpers.PrintTable(headings, data, o.Out); err != nil {
		return err
	}
	return nil
}
