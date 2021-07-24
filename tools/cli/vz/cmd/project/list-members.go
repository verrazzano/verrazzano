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
	PrintFlags *helpers.PrintFlags
}

func NewProjectListMembersOptions(streams genericclioptions.IOStreams) *ProjectListMembersOptions {
	return &ProjectListMembersOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdProjectListMembers(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewProjectListMembersOptions(streams)
	cmd := &cobra.Command{
		Use:   "list-members",
		Short: "list current members of the project",
		Long:  "list current members of the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listMembers(o, streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func listMembers(o *ProjectListMembersOptions, streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	projectName := args[0]
	projectClientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectName, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}

	// TODO : Logic for handling output flags
	// which group, version & kind are they?

	headings := []string{"NAME", "ROLE", "KIND"}
	data := [][]string{}
	for _, mSubject := range project.Spec.Template.Security.ProjectMonitorSubjects {
		rowData := []string{
			mSubject.Name,
			"ClusterRole/verrazzano-project-monitor",
			mSubject.Kind,
			// TODO : Look how to get the age here.
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
		// TODO : Look how to get the age here.
	}

	if len(data)==0{
		fmt.Fprintln(streams.Out,"no members present")
		return nil
	}else{
		err = helpers.PrintTable(headings, data, streams.Out)
		if err != nil {
			fmt.Fprintln(streams.ErrOut,err)
			return err
		}
	}

	return nil
}