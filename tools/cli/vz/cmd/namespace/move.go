package namespace

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type NamespaceMoveOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceMoveOptions(streams genericclioptions.IOStreams) *NamespaceMoveOptions {
	return &NamespaceMoveOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespaceMove(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "move namespace to a project",
		Long:  "move namespace to a project",
		//TODO : Look up how to write documentation for this command
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := moveNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func moveNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	// business logic
	// vz namespace move <projectname> <namespace anem>
	return nil
}
