// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	clustersclient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/typed/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectListOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args []string
	genericclioptions.IOStreams
}

func NewProjectListOptions(streams genericclioptions.IOStreams) *ProjectListOptions {
	return &ProjectListOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams: streams,
	}
}

func NewCmdProjectList(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewProjectListOptions(streams)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listProjects(args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func listProjects(args []string) error {
	config := pkg.GetKubeConfig()
	clientset, err := clustersclient.NewForConfig(config)
	if err != nil {
		fmt.Print("could not get the clientset")
	}

	// get a list of the projects
	projects, err := clientset.VerrazzanoProjects("verrazzano-mc").List(context.Background(), metav1.ListOptions{})
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
	helpers.PrintTable(headings, data)
	return nil
}
