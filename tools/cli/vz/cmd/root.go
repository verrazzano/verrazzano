// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/app"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/cluster"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/login"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/logout"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/project"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type RootOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewRootOptions(streams genericclioptions.IOStreams) *RootOptions {
	return &RootOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdRoot(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewRootOptions(streams)
	cmd := &cobra.Command{
		Use:   "vz",
		Short: "Verrazzano CLI",
		Long:  "Verrazzano CLI",
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(project.NewCmdProject(streams))
	cmd.AddCommand(cluster.NewCmdCluster(streams))
	cmd.AddCommand(app.NewCmdApp(streams))
	cmd.AddCommand(login.NewCmdLogin(streams))
	cmd.AddCommand(logout.NewCmdLogout(streams))
	return cmd
}
