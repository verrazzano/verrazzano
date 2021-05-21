// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args []string
	genericclioptions.IOStreams
}

func NewClusterOptions(streams genericclioptions.IOStreams) *ClusterOptions {
	return &ClusterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams: streams,
	}
}

func NewCmdCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClusterOptions(streams)
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Information about clusters",
		Long:  "Information about clusters",
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.AddCommand(NewCmdClusterList(streams))
	return cmd
}