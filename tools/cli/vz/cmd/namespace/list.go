package namespace

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type NamespaceListOptions struct {
	args []string
	genericclioptions.IOStreams
	PrintFlags *helpers.PrintFlags
}

func NewNamespaceListOptions(streams genericclioptions.IOStreams) *NamespaceListOptions {
	return &NamespaceListOptions{
		IOStreams:  streams,
		PrintFlags: helpers.NewGetPrintFlags(),
	}
}

func NewCmdNamespaceList(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewNamespaceListOptions(streams)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List namespaces",
		Long:    "List namespaces",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listNamespaces(o, streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func listNamespaces(o *NamespaceListOptions, streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	// business logic comes here
	return nil
}
