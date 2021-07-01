// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectListOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewProjectListOptions(streams genericclioptions.IOStreams) *ProjectListOptions {
	return &ProjectListOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdProjectList(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewProjectListOptions(streams)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listProjects(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func listProjects(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	if len(args) != 0 {
		fmt.Println("Got unexpected number of arguments")
		return nil
		// TODO : Check this return value and print statement again
	}
	//config := pkg.GetKubeConfig()
	//clientset, err := clustersclient.NewForConfig(config)
	clientset2, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Print("could not get the clientset")
	}

	//projects, err := clientset.VerrazzanoProjects("verrazzano-mc").List(context.Background(), metav1.ListOptions{})
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
