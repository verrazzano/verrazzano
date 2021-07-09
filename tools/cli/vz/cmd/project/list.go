// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectListOptions struct {
	args []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewProjectListOptions(streams genericclioptions.IOStreams) *ProjectListOptions {
	return &ProjectListOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdProjectList(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewProjectListOptions(streams)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List projects",
		Long:    "List projects",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listProjects(o, streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func listProjects(o *ProjectListOptions, streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {

	if len(args) != 0 {
		return errors.New("Got unexpected number of arguments, expected 0")
	}

	clientset2, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// get a list of the projects
	projects, err := clientset2.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	// check if the list is empty
	if len(projects.Items) == 0 {
		fmt.Println(helpers.NothingFound)
		return nil
	}

	// Output options was specified
	if len(*o.PrintFlags.OutputFormat) != 0 {
		// Set the Version and Kind before passing it as runtime object
		projects.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "clusters.verrazzano.io",
			Version: "v1alpha1",
			Kind:    "VerrazzanoProject",
		})

		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			fmt.Fprintln(streams.ErrOut, "Did not get a printer object")
			return err
		}
		err = printer.PrintObj(projects, o.Out)

		return err
	}

	// print out details of the projects
	headings := []string{"NAME", "AGE", "CLUSTERS", "NAMESPACES"}
	data := [][]string{}
	for _, project := range projects.Items {
		rowData := []string{
			project.Name,
			helpers.Age(project.CreationTimestamp),
			helpers.FormatStringSlice(func() []string {
				result := []string{}
				for _, x := range project.Spec.Placement.Clusters {
					result = append(result, x.Name)
				}
				return result
			}()),
			helpers.FormatStringSlice(func() []string {
				result := []string{}
				for _, x := range project.Spec.Template.Namespaces {
					result = append(result, x.Metadata.Name)
				}
				return result
			}()),
		}
		data = append(data, rowData)
	}

	// print out the data
	helpers.PrintTable(headings, data, streams.Out)
	return nil
}
