// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var projectNamespaces []string
var projectPlacement []string

type ProjectAddOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectAddOptions(streams genericclioptions.IOStreams) *ProjectAddOptions {
	return &ProjectAddOptions{
		IOStreams: streams,
	}
}

func NewCmdProjectAdd(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add name",
		Short: "Add a project",
		Long:  "Add a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := addProject(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&projectNamespaces, "namespaces", "n", []string{}, "List of namespaces to include in the project")
	cmd.Flags().StringSliceVarP(&projectPlacement, "placement", "p", []string{"local"}, "List of clusters this project will be placed in")
	return cmd
}

func addProject(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	projectName := args[0]

	// if no namespace was provided, default to a single namespace
	// with the same name as the project itself
	if len(projectNamespaces) == 0 {
		projectNamespaces = []string{projectName}
	}

	// prepare the project resource
	project := v1alpha1.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectName,
			Namespace: "verrazzano-mc",
		},
		Spec: v1alpha1.VerrazzanoProjectSpec{
			Template: v1alpha1.ProjectTemplate{
				Namespaces: func() []v1alpha1.NamespaceTemplate {
					var namespaces []v1alpha1.NamespaceTemplate
					for _, v := range projectNamespaces {
						namespaces = append(namespaces, v1alpha1.NamespaceTemplate{
							Metadata: metav1.ObjectMeta{
								Name: v,
							},
						})
					}
					return namespaces
				}(),
			},
			Placement: v1alpha1.Placement{
				Clusters: func() []v1alpha1.Cluster {
					var placements []v1alpha1.Cluster
					for _, v := range projectPlacement {
						placements = append(placements, v1alpha1.Cluster{
							Name: v,
						})
					}
					return placements
				}(),
			},
		},
	}

	// connect to the cluster
	clientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// create the project resource in the cluster
	_, err = clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Create(context.Background(), &project, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	fmt.Fprintln(streams.Out, "project/"+projectName+" added")
	return nil
}
