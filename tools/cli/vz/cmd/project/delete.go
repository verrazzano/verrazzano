// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	clustersclient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/typed/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectDeleteOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewProjectDeleteOptions(streams genericclioptions.IOStreams) *ProjectDeleteOptions {
	return &ProjectDeleteOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdProjectDelete(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewProjectDeleteOptions(streams)
	cmd := &cobra.Command{
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
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
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
