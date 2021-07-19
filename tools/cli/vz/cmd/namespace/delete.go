package namespace

import (
	"context"
	"fmt"
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
	err := clientset.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	projectClientset, err2 := kubernetesInterface.NewProjectClientSet()
	if err2 != nil {
		fmt.Fprintln(streams.ErrOut, err2)
		return err2
	}

	projects, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").List(context.Background(), metav1.ListOptions{})
	for _, project := range projects.Items {
		for i, namespace := range project.Spec.Template.Namespaces {
			if namespace.Metadata.Name == nsName {
				project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces[:i], project.Spec.Template.Namespaces[i+1:]...)
			}
		}

		_, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), &project, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
	}
	fmt.Fprintln(streams.Out, "namepspace "+`"`+nsName+`"`+" deleted")
	return nil

}
