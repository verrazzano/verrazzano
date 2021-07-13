package namespace

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var projectID, description []string

type NamespaceAddOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceAddOptions(streams genericclioptions.IOStreams) *NamespaceAddOptions {
	return &NamespaceAddOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespaceAdd(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add namespace",
		Short: "Add a namespace",
		Long:  "Add a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := AddNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&description, "description", []string{}, "description about the namespace")
	cmd.Flags().StringSliceVarP(&projectID, "project-id", "p", []string{}, "ID of project this namespace belongs to")
	return cmd
}

func AddNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	// Business logic for creating namespace here.
	return nil
}
