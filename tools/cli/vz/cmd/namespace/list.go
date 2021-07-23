// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type NamespaceListOptions struct {
	args []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewNamespaceListOptions(streams genericclioptions.IOStreams) *NamespaceListOptions {
	return &NamespaceListOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdNamespaceList(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewNamespaceListOptions(streams)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List namespaces",
		Long:    "List namespaces",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listNamespace(o, streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func listNamespace(o *NamespaceListOptions, streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	clientset := kubernetesInterface.NewClientSet()
	collection, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	})
	if err != nil {
		fmt.Fprintln(streams.Out, err)
		return err
	}

	// Output options was specified
	if len(*o.PrintFlags.OutputFormat) != 0 {
		// Set the Version and Kind before passing it as runtime object
		collection.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Version: "v1",
			Kind:    "namespaces",
		})
		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			fmt.Fprintln(streams.ErrOut, "Did not get a printer object")
			return err
		}
		err = printer.PrintObj(collection, o.Out)
		return err
	}

	// how to get description
	headings := []string{"NAME", "STATE", "PROJECT", "AGE"}
	data := [][]string{}
	for _, ns := range collection.Items {
		var projectName string
		labels := ns.GetLabels()
		// if namespace is a verrazzano namespace, display namespace
		for s, s2 := range labels {
			if s == "verrazzano-managed" && s2 == "true" {
				projectName = labels["verrazzano/projectName"]
				rowData := []string{
					ns.Name,
					string(ns.Status.Phase),
					projectName,
					helpers.Age(ns.CreationTimestamp),
				}
				data = append(data, rowData)
			}
		}
	}
	if len(data) == 0 {
		fmt.Fprintln(streams.Out, "no verrazzano namespaces exist")
		return nil
	}
	err = helpers.PrintTable(headings, data, streams.Out)
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}

	return nil
}
