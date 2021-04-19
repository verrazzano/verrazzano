// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersclient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/typed/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var projectNamespaces []string
var projectPlacement []string

func init() {
	projectAddCmd.Flags().StringSliceVarP(&projectNamespaces, "namespaces", "n", []string{}, "List of namespaces to include in the project")
	projectAddCmd.Flags().StringSliceVarP(&projectPlacement, "placement", "p", []string{"local"}, "List of clusters this project will be placed in")
	projectCmd.AddCommand(projectAddCmd)
}

var projectAddCmd = &cobra.Command{
	Use: "add name",
	Short: "Add a project",
	Long: "Add a project",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := addProject(args); err != nil {
			return err
		}
		return nil
	},
}

func addProject(args []string) error {
	projectName := args[0]

	// if no namespace was provided, default to a single namespace
	// with the same name as the project itself
	if len(projectNamespaces) == 0 {
		projectNamespaces = []string{ projectName }
	}

	// prepare the project resource
	project := v1alpha1.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
			Namespace: "verrazzano-mc",
		},
		Spec:       v1alpha1.VerrazzanoProjectSpec{
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
	config := pkg.GetKubeConfig()
	clientset, err := clustersclient.NewForConfig(config)
	if err != nil {
		fmt.Print("could not get the clientset")
	}

	// create the project resource in the cluster
	_, err = clientset.VerrazzanoProjects("verrazzano-mc").Create(context.Background(), &project, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Println("project created")
	return nil
}