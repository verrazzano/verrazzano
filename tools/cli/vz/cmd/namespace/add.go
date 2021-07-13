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
	// TODO : Confirm if project flag is mandatory. Make the project flag mandatory
	// throw an error if project is not specified
	/*if len(projectID[0])==0{
		fmt.Fprintln(streams.ErrOut,"project flag is mandatory")
	}*/
	return cmd
}

func AddNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	/*if len(projectID[0])==0{
		//fmt.Fprintln(streams.ErrOut,"project flag is mandatory")
		return errors.New("project flag is mandatory")
	}*/
	nsName := args[0]

	//preparing namespace resource
	// TODO : Should i implement creation timestamp? it's empty when viewed through project command.
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
