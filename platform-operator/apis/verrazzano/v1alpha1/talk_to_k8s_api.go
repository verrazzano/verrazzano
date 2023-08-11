// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetVerrazzanoV1Alpha1 returns a v1alpha1 Verrazzano struct.
// This function internally uses v1beta1 Verrazzano to ask Kubernetes API server for the VZ resource.
func GetVerrazzanoV1Alpha1(ctx context.Context, client client.Client, name types.NamespacedName) (*Verrazzano, error) {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := client.Get(ctx, name, vzV1Beta1); err != nil {
		return nil, err
	}
	vzV1Alpha1 := &Verrazzano{}
	if err := vzV1Alpha1.ConvertFrom(vzV1Beta1); err != nil {
		return nil, err
	}
	return vzV1Alpha1, nil
}

// UpdateVerrazzanoV1Alpha1 takes in a v1alpha1 Verrazzano struct and sends an update request to the K8s API server
// for that resource. This function internally converts the Verrazzano to v1beta1 before sending the update request.
func UpdateVerrazzanoV1Alpha1(ctx context.Context, client client.Client, vzV1Alpha1 *Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vzV1Alpha1.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	if err := client.Update(ctx, vzV1Beta1); err != nil {
		return err
	}
	return nil
}
