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

var projectID = []string{""}
var description []string

type NamespaceCreateOptions struct {
	args []string
	genericclioptions.IOStreams
}

func NewNamespaceCreateOptions(streams genericclioptions.IOStreams) *NamespaceCreateOptions {
	return &NamespaceCreateOptions{
		IOStreams: streams,
	}
}

func NewCmdNamespaceCreate(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create namespace",
		Short: "create a namespace",
		Long:  "Create a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := createNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&description, "description", []string{}, "description about the namespace")
	cmd.Flags().StringSliceVarP(&projectID, "project-id", "p", []string{}, "ID of project this namespace belongs to")
	// TODO : Throw a more graceful error when project flag is not specified.
	return cmd
}

func createNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	nsName := args[0]

	//preparing namespace resource
	// TODO : Isolate error in project about createTimeStamp, or pass it here.
	namespace := v1alpha1.NamespaceTemplate{
		Metadata: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	// fetching the clientset
	clientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
	}

	// fetching the project
	project, err := clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectID[0], metav1.GetOptions{})
	if err != nil {
		return err
	}

	// appending the namespace to the project
	var isDuplicate bool
	// TODO : Parse the project's namespaces to check if it already exists. Look how to do it.
	for _, template := range project.Spec.Template.Namespaces {
		if template.Metadata.Name == projectID[0] {
			isDuplicate = true
			fmt.Fprintln(streams.Out, "duplicate namespace")
		}
	}

	// if the project doesn't already contain the namespace
	if !isDuplicate {
		// adding the namespace to the project
		project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces, namespace)

		// updating the project
		if _, err := clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}
