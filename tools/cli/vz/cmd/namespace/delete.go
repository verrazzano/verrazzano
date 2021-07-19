package namespace

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type NamespaceDeleteOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceDeleteOptions(streams genericclioptions.IOStreams) *NamespaceDeleteOptions {
	return &NamespaceDeleteOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespaceDelete(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete namespace",
		Long:  "delete namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func deleteNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	nsName := args[0]

	// getting the clientset
	clientset := kubernetesInterface.NewClientSet()
	err2 := clientset.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{})
	if err2 != nil {
		return err2
	}
	return nil

}
