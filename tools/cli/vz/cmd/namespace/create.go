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
	cmd.Flags().StringSliceVarP(&projectID, "project-id", "p", []string{}, "ID of project this namespace belongs to")
	// TODO : Throw a more graceful error when project flag is not specified - give a default project?
	return cmd
}

func createNamespace(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	nsName := args[0]

	//preparing namespace resources
	// TODO : Isolate error in project about createTimeStamp, or pass it here.
	
	// old namespace
	oldNamespace := v1alpha1.NamespaceTemplate{
		Metadata: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	
	newNamespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	// fetching the clientsets
	projectClientset, err := kubernetesInterface.NewProjectClientSet()
	if err!=nil{
		fmt.Fprintln(streams.ErrOut,err)
	}
	clustersClientset:= kubernetesInterface.NewClientSet()
	newns, err := clustersClientset.CoreV1().Namespaces().Create(context.Background(),&newNamespace,metav1.CreateOptions{})
	
	// fetching the project
	project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectID[0], metav1.GetOptions{})
	//project, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(),projectID[0],metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces,newNamespace.Spec.)

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
		project.Spec.Template.Namespaces = append(project.Spec.Template.Namespaces, oldNamespace)

		// updating the project
		if _, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), project, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}
