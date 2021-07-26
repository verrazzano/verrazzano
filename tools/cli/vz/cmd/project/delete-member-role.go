// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectDeleteMemberRoleOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectDeleteMemberRoleOptions(streams genericclioptions.IOStreams) *ProjectDeleteMemberRoleOptions {
	return &ProjectDeleteMemberRoleOptions{
		IOStreams: streams,
	}
}

func NewCmdProjectDeleteMemberRole(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-member-role -p PROJECT MEMBER",
		Short: "Delete a member role from a project",
		Long:  "Delete a member role from a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteMemberRole(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&projectName, "project-name", "p", []string{}, "project to delete member-role from")
	cmd.MarkFlagRequired("project-name")
	return cmd
}

func deleteMemberRole(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	isfound := -1
	mName := args[0]
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
	var index int
	for i, subject := range project.Spec.Template.Security.ProjectMonitorSubjects {
		if subject.Name == mName {
			isfound = 1 // if isfound is set to 1, it is project-monitor-role
			index = i
			break
		}
	}

	for i, subject := range project.Spec.Template.Security.ProjectAdminSubjects {
		if subject.Name == mName {
			isfound = 2 // if isfound is set to 2, it is project-admin-role
			index = i
			break
		}
	}

	if isfound == -1 {
		fmt.Fprintln(streams.ErrOut, errors.New("Member not found"))
	} else {
		if isfound == 1 {
			project.Spec.Template.Security.ProjectMonitorSubjects = append(project.Spec.Template.Security.ProjectMonitorSubjects[:index], project.Spec.Template.Security.ProjectMonitorSubjects[index+1:]...)
		} else {
			project.Spec.Template.Security.ProjectAdminSubjects = append(project.Spec.Template.Security.ProjectAdminSubjects[:index], project.Spec.Template.Security.ProjectAdminSubjects[index+1:]...)
		}

		_, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
		if isfound == 1 {
			fmt.Fprintln(streams.Out, `ProjectMonitor Subject "`+mName+`" deleted`)
		} else {
			fmt.Fprintln(streams.Out, `ProjectAdmin Subject "`+mName+`" deleted`)
		}

	}
	return nil
}
