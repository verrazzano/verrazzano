// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewProjectOptions(streams genericclioptions.IOStreams) *ProjectOptions {
	return &ProjectOptions{
		IOStreams: streams,
	}
}

func NewCmdProject(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Work with projects",
		Long:  "Work with projects",
	}
	cmd.AddCommand(NewCmdProjectList(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdProjectAdd(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdProjectDelete(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdProjectGet(streams, kubernetesInterface))
	return cmd
}
