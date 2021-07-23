// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type NamespaceOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceOptions(streams genericclioptions.IOStreams) *NamespaceOptions {
	return &NamespaceOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespace(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespace",
		Short:   "Work with namespaces",
		Long:    "Work with namespaces",
		Aliases: []string{"ns"},
		// TODO : Description needs to be rewritten
	}
	cmd.AddCommand(NewCmdNamespaceCreate(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdNamespaceList(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdNamespaceMove(streams, kubernetesInterface))
	cmd.AddCommand(NewCmdNamespaceDelete(streams, kubernetesInterface))
	return cmd
}
