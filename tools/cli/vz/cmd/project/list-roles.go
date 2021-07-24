// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectListRolesOptions struct {
	args []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewProjectListRolesOptions(streams genericclioptions.IOStreams) *ProjectListRolesOptions {
	return &ProjectListRolesOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdProjectListRoles(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewProjectListRolesOptions(streams)
	cmd := &cobra.Command{
		Use:   "list-roles",
		Short: "List available roles for a project",
		Long:  "List available roles for a project",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listProjectRoles(o, streams, kubernetesInterface); err != nil {
				fmt.Fprintln(streams.ErrOut, err)
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func listProjectRoles(o *ProjectListRolesOptions, streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) error {
	k8Clientset := kubernetesInterface.NewClientSet()
	roles, err := k8Clientset.RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	if len(roles.Items) == 0 {
		fmt.Fprintln(streams.Out, helpers.NothingFound)
		return nil
	}

	// Output options was specified
	if len(*o.PrintFlags.OutputFormat) != 0 {
		// Set the Version and Kind before passing it as runtime object
		roles.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
			Kind:    "ClusterRole",
		})

		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			fmt.Fprintln(streams.ErrOut, "Did not get a printer object")
			return err
		}
		err = printer.PrintObj(roles, o.Out)

		return err
	}

	headings := []string{"NAME", "AGE"}
	data := [][]string{}
	for _, role := range roles.Items {
		if role.GetName() == "verrazzano-project-admin" || role.GetName() == "verrazzano-project-monitor" {
			rowData := []string{
				role.GetName(),
				helpers.Age(role.GetCreationTimestamp()),
			}
			data = append(data, rowData)
		}
	}
	helpers.PrintTable(headings, data, streams.Out)

	return nil
}
