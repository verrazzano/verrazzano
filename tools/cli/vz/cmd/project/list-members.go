// Copyright (c) 2021, Oracle and/or its affiliates.
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

type ProjectListMembersOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectListMembersOptions(streams genericclioptions.IOStreams) *ProjectListMembersOptions {
	return &ProjectListMembersOptions{
		IOStreams: streams,
	}
}

func NewCmdProjectListMembers(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-members -p PROJECT",
		Short: "list current members of the project",
		Long:  "list current members of the project",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listMembers(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&projectName, "project-name", "p", []string{}, "project to display member-roles of")
	cmd.MarkFlagRequired("project-name")
	return cmd
}

func listMembers(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	projectClientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectName[0], metav1.GetOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}

	headings := []string{"NAME", "ROLE", "KIND"}
	data := [][]string{}
	for _, mSubject := range project.Spec.Template.Security.ProjectMonitorSubjects {
		rowData := []string{
			mSubject.Name,
			"ClusterRole/verrazzano-project-monitor",
			mSubject.Kind,
		}
		data = append(data, rowData)
	}
	for _, aSubject := range project.Spec.Template.Security.ProjectAdminSubjects {
		rowData := []string{
			aSubject.Name,
			"ClusterRole/verrazzano-project-admin",
			aSubject.Kind,
		}
		data = append(data, rowData)
	}

	if len(data) == 0 {
		fmt.Fprintln(streams.Out, "no members present")
	} else {
		err = helpers.PrintTable(headings, data, streams.Out)
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
	}
	return nil
}
