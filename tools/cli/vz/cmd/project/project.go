// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProjectOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args []string
	genericclioptions.IOStreams
}

func NewProjectOptions(streams genericclioptions.IOStreams) *ProjectOptions {
	return &ProjectOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams: streams,
	}
}

func NewCmdProject(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewProjectOptions(streams)
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Work with projects",
		Long:  "Work with projects",
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(NewCmdProjectList(streams))
	cmd.AddCommand(NewCmdProjectAdd(streams))
	cmd.AddCommand(NewCmdProjectDelete(streams))
	return cmd
}