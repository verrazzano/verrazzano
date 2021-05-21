// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/app/helidon"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type AppOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewAppOptions(streams genericclioptions.IOStreams) *AppOptions {
	return &AppOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdApp(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewAppOptions(streams)
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Information about applications",
		Long:  "Information about applications",
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(helidon.NewCmdAppHelidon(streams))
	return cmd
}
