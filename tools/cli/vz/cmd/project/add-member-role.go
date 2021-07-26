// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var projectName []string

type ProjectAddMemberRoleOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectAddMemberRoleOptions(streams genericclioptions.IOStreams) *ProjectAddMemberRoleOptions {
	return &ProjectAddMemberRoleOptions{
		IOStreams: streams,
	}
}

func NewCmdProjectAddMemberRole(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member-role -p PROJECT MEMBER MEMBER-ROLE",
		Short: "Add a member role to a project",
		Long:  "Add a member role to a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := addMemberRole(streams, args, kubernetesInterface); err != nil {
				fmt.Fprintln(streams.ErrOut, err)
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&projectName, "project-name", "p", []string{}, "project to add member-role")
	//cmd.MarkFlagRequired("project-name")
	return cmd
}

func addMemberRole(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	mName := args[0]
	mrole := args[1]
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

	newSubject := v1.Subject{
		Kind: "User",
		Name: mName,
	}
	if mrole == "verrazzano-project-admin" {
		project.Spec.Template.Security.ProjectAdminSubjects = append(project.Spec.Template.Security.ProjectAdminSubjects, newSubject)
	} else {
		project.Spec.Template.Security.ProjectMonitorSubjects = append(project.Spec.Template.Security.ProjectMonitorSubjects, newSubject)
	}

	_, err = projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	fmt.Fprintln(streams.Out, `member role "`+mrole+`" added`)
	return nil
}
