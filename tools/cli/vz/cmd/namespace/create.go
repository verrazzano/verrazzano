// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var projectName []string
var description []string
var projectDeclared bool

type NamespaceCreateOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceCreateOptions(streams genericclioptions.IOStreams) *NamespaceCreateOptions {
	return &NamespaceCreateOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespaceCreate(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create NAMESPACE",
		Short: "Create a namespace",
		Long:  "Create a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := createNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&projectName, "project-name", "p", []string{""}, "Name of project this namespace belongs to")
	// checking if project-name flag is passed.
	projectDeclared = cmd.Flags().Changed("project-name")
	return cmd
}

func createNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	nsName := args[0]
	// adding label to register namespace as verrazzano namespace
	vzLabel := make(map[string]string)
	vzLabel["verrazzano-managed"] = "true"

	// if project flag is passed
	if projectDeclared {
		projectClientset, err := kubernetesInterface.NewProjectClientSet()
		project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectName[0], metav1.GetOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}

		// preparing the namespace
		vzLabel["verrazzano/projectName"] = projectName[0]
		newNsTemplate := v1alpha1.NamespaceTemplate{
			Metadata: metav1.ObjectMeta{
				Name:              nsName,
				CreationTimestamp: metav1.Now(),
				Labels:            vzLabel,
			},
		}
		// adding the namespace to the project
		project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces, newNsTemplate)
		if _, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{}); err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
	} else {
		// preparing the namespace object
		newNamespace := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              nsName,
				CreationTimestamp: metav1.Now(),
				Labels:            vzLabel,
			},
		}

		// fetching the clientsets
		clustersClientset := kubernetesInterface.NewClientSet()
		_, err := clustersClientset.CoreV1().Namespaces().Create(context.Background(), &newNamespace, metav1.CreateOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
	}

	fmt.Fprintln(streams.Out, "namespace/"+nsName+" created")
	return nil
}
