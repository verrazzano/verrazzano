// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectGetOptions struct {
	args []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewProjectGetOptions(streams genericclioptions.IOStreams) *ProjectGetOptions {
	return &ProjectGetOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdProjectGet(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewProjectGetOptions(streams)
	cmd := &cobra.Command{
		Use:   "get name",
		Short: "display a project's details",
		Long:  "display a project's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getProject(o, streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func getProject(o *ProjectGetOptions, streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	var projectName = args[0]

	// if projectName isn't provided
	if len(projectName) == 0 {
		fmt.Fprintln(streams.ErrOut, "project name needs to be provided")
		return nil
	}

	// connect to the cluster
	clientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// fetching the project resource from the cluster
	var projectDetails *v1alpha1.VerrazzanoProject
	projectDetails, err = clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Output options was specified
	if len(*o.PrintFlags.OutputFormat) != 0 {
		// Set the Version and Kind before passing it as runtime object
		projectDetails.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "clusters.verrazzano.io",
			Version: "v1alpha1",
			Kind:    "VerrazzanoProject",
		})

		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			fmt.Fprintln(streams.ErrOut, "Did not get a printer object")
			return err
		}
		err = printer.PrintObj(projectDetails, o.Out)

		return err
	}

	// print out details of the project
	headings := []string{"NAME", "AGE", "CLUSTERS", "NAMESPACES"}
	data := [][]string{}
	projectData := []string{
		projectDetails.Name,
		helpers.Age(projectDetails.CreationTimestamp),
		helpers.FormatStringSlice(func() []string {
			result := []string{}
			for _, x := range projectDetails.Spec.Placement.Clusters {
				result = append(result, x.Name)
			}
			return result
		}()),
		helpers.FormatStringSlice(func() []string {
			result := []string{}
			for _, x := range projectDetails.Spec.Template.Namespaces {
				result = append(result, x.Metadata.Name)
			}
			return result
		}()),
	}
	data = append(data, projectData)

	helpers.PrintTable(headings, data, streams.Out)
	return nil
}
