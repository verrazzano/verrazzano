// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"errors"
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName   = "status"
	namespaceFlag = "namespace"
	nameFlag      = "name"
)

var namespace string
var name string

func NewCmdStatus(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: "Status of the Verrazzano install and access endpoints",
		Long:  "Status of the Verrazzano install and access endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmdStatus(cmd, args, vzHelper)
		},
	}

	// Add flags specific to this command and its sub-commands
	cmd.PersistentFlags().StringVarP(&namespace, namespaceFlag, "n", "default", "The namespace of the Verrazzano resource")
	cmd.PersistentFlags().StringVar(&name, nameFlag, "", "The name of the Verrazzano resource")

	return cmd
}

// runCmdStatus - run the "vz status" command
func runCmdStatus(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	fmt.Fprintln(vzHelper.GetOutputStream(), fmt.Sprintf("The name is %s in namespace %s", name, namespace))

	client, err := vzHelper.GetClient()
	if err != nil {
		return err
	}

	// Get the VZ resource
	vz := vzapi.Verrazzano{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to find Verrazzano with name %s in namespace %s: %v", name, namespace, err.Error()))
	}

	// Report the status information
	fmt.Fprintln(vzHelper.GetOutputStream(), fmt.Sprintf("Version %s is installed", vz.Status.Version))

	return nil
}
