// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetVerrazzano gets the Verrazzano CR
func (c Client) GetVerrazzano(namespace string, name string) (*vzapi.Verrazzano, error) {
	cr, err := c.VzClient.Verrazzanos(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return cr, err
}
