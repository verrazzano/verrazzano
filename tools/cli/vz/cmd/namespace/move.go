// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

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
		Use:   "move NAMESPACE PROJECT",
		Short: "move namespace to a project",
		Long:  "move a namespace to a different project",
		Args:  cobra.ExactArgs(2),
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
	nsName := args[0]
	projectName := args[1]

	projectClientset, err := kubernetesInterface.NewProjectClientSet()
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	clustersClientset := kubernetesInterface.NewClientSet()

	destProject, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), projectName, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	namespace, err := clustersClientset.CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}

	// making namespace a verrazzano namespace
	nsLabels := namespace.GetLabels()
	if len(nsLabels)==0{
		nsLabels = make(map[string]string)
	}
		nsLabels["verrazzano-managed"] = "true"
	namespace.SetLabels(nsLabels)

	// if namespace is already existing in a vz project
	if nsLabels["verrazzano/projectName"]!="" {
		srcProjectName := nsLabels["verrazzano/projectName"]
		// remove namespace from srcProject
		srcProject, err := projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.Background(), srcProjectName, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintln(streams.ErrOut, err)
			return err
		}
		delIndex := 0
		// finding index of namespace to delete in the source project file
		for i, nsTemplate := range srcProject.Spec.Template.Namespaces {
			if nsTemplate.Metadata.Name == nsName {
				delIndex = i
				break
			}
		}

		// deleting namespace from source project file
		srcProject.Spec.Template.Namespaces = append(srcProject.Spec.Template.Namespaces[:delIndex], srcProject.Spec.Template.Namespaces[delIndex+1:]...)

		// updating the project.
		srcProject, err = projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), srcProject, metav1.UpdateOptions{})
		if err != nil {
		//	fmt.Fprintln(streams.ErrOut, err)
			return err
		}
	}

	nsLabels["verrazzano/projectName"] = destProject.GetName()
	namespace.SetLabels(nsLabels)

	// add namespace to destProject
	nsTemplate := v1alpha1.NamespaceTemplate{
		Metadata: metav1.ObjectMeta{
			Name:              namespace.GetName(),
			CreationTimestamp: namespace.GetCreationTimestamp(),
			Labels:            namespace.GetLabels(),
		},
	}
	destProject.Spec.Template.Namespaces = append(destProject.Spec.Template.Namespaces, nsTemplate)
	_, err = projectClientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Update(context.Background(), destProject, metav1.UpdateOptions{})
	if err != nil {
		fmt.Fprintln(streams.ErrOut, err)
		return err
	}
	fmt.Fprintln(streams.Out,`"`+nsName+`" namespace moved to "`+destProject.GetName()+`" project`)
	return nil
}
