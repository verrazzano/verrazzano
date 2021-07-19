package namespace

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var projectID []string
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
		Short: "Create a namespace",
		Long:  "Create a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := createNamespace(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&description, "description", []string{}, "Description about the namespace")
	cmd.Flags().StringSliceVarP(&projectID, "project-id", "p", []string{""}, "ID of project this namespace belongs to")
	// TODO : Throw a more graceful error when project flag is not specified - give a default project?
	return cmd
}

func createNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	nsName := args[0]

	// adding label to register namespace as verrazzano namespace
	vzLabel := make(map[string]string)
	vzLabel["verrazzano-managed"] = "true"

	// preparing the namespace object
	newNamespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              nsName,
			CreationTimestamp: metav1.Now(),
			Labels:            vzLabel,
		},
	}

	// fetching the clientsets
	clustersClientset := kubernetesInterface.NewClientSet()
	_, err := clustersClientset.CoreV1().Namespaces().Create(context.Background(), &newNamespace, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}

	// if project flag is passed
	if projectID[0] != "" {
		isDuplicate := false
		projectClientset, err := kubernetesInterface.NewProjectClientSet()
		project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectID[0], metav1.GetOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}

		// checking if project already contains the namespace.
		for _, projectNamespaces := range project.Spec.Template.Namespaces {
			if projectNamespaces.Metadata.Name == projectID[0] {
				isDuplicate = true
			}
		}
		if !isDuplicate {
			// adding the namespace to the project
			newNsTemplate := v1alpha1.NamespaceTemplate{
				Metadata: metav1.ObjectMeta{
					Name:              newNamespace.GetName(),
					CreationTimestamp: newNamespace.GetCreationTimestamp(),
				},
			}
			project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces, newNsTemplate)
			// updating the project
			if _, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{}); err != nil {
				fmt.Fprintln(streams.ErrOut, err)
				return err
			}
		}
	}
	fmt.Fprintln(streams.Out, "namespace/"+nsName+" created")
	return nil
}
