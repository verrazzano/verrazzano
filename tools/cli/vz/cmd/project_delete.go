// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	clustersclient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/typed/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	projectCmd.AddCommand(projectDeleteCmd)
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete name",
	Short: "Delete a project",
	Long:  "Delete a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := deleteProject(args); err != nil {
			return err
		}
		return nil
	},
}

func deleteProject(args []string) error {
	projectName := args[0]

	// connect to the cluster
	config := pkg.GetKubeConfig()
	clientset, err := clustersclient.NewForConfig(config)
	if err != nil {
		fmt.Print("could not get the clientset")
	}

	// delete the project
	err = clientset.VerrazzanoProjects("verrazzano-mc").Delete(context.Background(), projectName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	fmt.Println("project deleted")
	return nil
}
