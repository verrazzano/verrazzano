// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VZHelper interface {
	GetOutputStream() io.Writer
	GetErrorStream() io.Writer
	GetInputStream() io.Reader
	GetClient(cmd *cobra.Command) (client.Client, error)
}

// FindVerrazzanoResource - find the single Verrazzano resource
func FindVerrazzanoResource(client client.Client) (*vzapi.Verrazzano, error) {

	vzList := vzapi.VerrazzanoList{}
	err := client.List(context.TODO(), &vzList)
	if err != nil {
		return nil, err
	}
	if len(vzList.Items) == 0 {
		return nil, fmt.Errorf("Failed to find any Verrazzano resources")
	}
	if len(vzList.Items) != 1 {
		return nil, fmt.Errorf("Expected to only find one Verrazzano resource, but found %d", len(vzList.Items))
	}
	return &vzList.Items[0], nil
}
