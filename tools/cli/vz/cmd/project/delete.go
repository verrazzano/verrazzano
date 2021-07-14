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

type ProjectDeleteOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectDeleteOptions(streams genericclioptions.IOStreams) *ProjectDeleteOptions {
	return &ProjectDeleteOptions{
		IOStreams: streams,
	}
}

// preserving this structure for any future flags.

func NewCmdProjectDelete(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete name",
		Short: "Delete a project",
		Long:  "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteProject(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func deleteProject(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	projectName := args[0]

	// connect to the cluster
	clientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// delete the project
	err = clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Delete(context.Background(), projectName, metav1.DeleteOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	fmt.Fprintln(streams.Out, "project/"+projectName+" deleted")
	return nil
}
