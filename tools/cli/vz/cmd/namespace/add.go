package namespace

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	nsName := args[0]

	//preparing namespace resource
	// TODO : Check this namespace implementation

	/*namespace := corev1.Namespace{
		ObjectMeta : metav1.ObjectMeta{
			Name: nsName,
		},
	}*/
	ns := v1alpha1.NamespaceTemplate{Metadata: metav1.ObjectMeta{Name: nsName}}

	clientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// assuming project passed is project1
	// TODO : Check this logic later

	project, err := clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectID[0], metav1.GetOptions{})
	if err != nil {
		return err
	}

	project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces, ns)

	if _, err := clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
