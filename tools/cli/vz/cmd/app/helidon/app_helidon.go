// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type AppHelidonOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewAppHelidonOptions(streams genericclioptions.IOStreams) *AppHelidonOptions {
	return &AppHelidonOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdAppHelidon(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewAppHelidonOptions(streams)
	cmd := &cobra.Command{
		Use:   "helidon",
		Short: "Work with Helidon applications",
		Long:  "Work with Helidon applications",
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(NewCmdAppHelidonCreate(streams))
	cmd.AddCommand(NewCmdAppHelidonList(streams))
	return cmd
}
