// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CommandName   = "status"
	namespaceFlag = "namespace"
	nameFlag      = "name"
)

var namespace string
var name string

func NewCmdStatus(k8s helpers.Kubernetes) *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: "Status of the Verrazzano install and access endpoints",
		Long:  "Status of the Verrazzano install and access endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmdStatus(cmd, args, k8s)
		},
	}

	// Add command specific flags
	cmd.LocalFlags().StringVarP(&namespace, namespaceFlag, "n", "default", "The namespace of the Verrazzano resource")
	cmd.LocalFlags().StringVar(&name, nameFlag, "", "The name of the Verrazzano resource")

	return cmd
}

func runCmdStatus(cmd *cobra.Command, args []string, k8s helpers.Kubernetes) error {
	fmt.Println(fmt.Sprintf("The name is %s in namespace %s", name, namespace))

	clientSet, err := k8s.NewVerrazzanoClientSet()
	if err != nil {
		return err
	}

	// Get the VZ resource
	verrazzano, err := clientSet.VerrazzanoV1alpha1().Verrazzanos("foo").Get(context.TODO(), "foo", metav1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Sprintf("The name is %s in namespace %s", verrazzano.Name, verrazzano.Namespace)

	return nil
}
