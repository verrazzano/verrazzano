// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetV1Alpha1 returns a v1alpha1 Verrazzano struct.
// This function internally uses v1beta1 Verrazzano to ask Kubernetes API server for the VZ resource.
func GetV1Alpha1(ctx context.Context, client client.Client, name types.NamespacedName) (*v1alpha1.Verrazzano, error) {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := client.Get(ctx, name, vzV1Beta1); err != nil {
		return nil, err
	}
	vzV1Alpha1 := &v1alpha1.Verrazzano{}
	if err := vzV1Alpha1.ConvertFrom(vzV1Beta1); err != nil {
		return nil, err
	}
	return vzV1Alpha1, nil
}

// ListV1Alpha1 returns a v1alpha1 VerrazzanoList.
// This function internally uses v1beta1 Verrazzano to talk to the K8s API server.
func ListV1Alpha1(ctx context.Context, client client.Client) (*v1alpha1.VerrazzanoList, error) {
	vzListV1Beta1 := &v1beta1.VerrazzanoList{}
	if err := client.List(ctx, vzListV1Beta1); err != nil {
		return nil, err
	}
	vzListV1Alpha1, err := convertV1Beta1ListToV1Alpha1(vzListV1Beta1)
	if err != nil {
		return nil, err
	}
	return vzListV1Alpha1, nil
}

// convertV1Beta1ListToV1Alpha1 converts the VerrazzanoList from v1beta1 to v1alpha1
func convertV1Beta1ListToV1Alpha1(vzListV1Beta1 *v1beta1.VerrazzanoList) (*v1alpha1.VerrazzanoList, error) {
	vzListV1Alpha1 := &v1alpha1.VerrazzanoList{}
	for _, vzV1Beta1 := range vzListV1Beta1.Items {
		vzV1Beta1 := vzV1Beta1
		vzV1Alpha1 := &v1alpha1.Verrazzano{}
		if err := vzV1Alpha1.ConvertFrom(&vzV1Beta1); err != nil {
			return nil, err
		}
		vzListV1Alpha1.Items = append(vzListV1Alpha1.Items, *vzV1Alpha1)
	}
	return vzListV1Alpha1, nil
}

// UpdateV1Alpha1 takes in a v1alpha1 Verrazzano struct and sends an update request to the K8s API server
// for that resource. This function internally converts the Verrazzano to v1beta1 before sending the update request.
func UpdateV1Alpha1(ctx context.Context, client client.Client, vzV1Alpha1 *v1alpha1.Verrazzano, updateOpts ...client.UpdateOption) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vzV1Alpha1.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	if err := client.Update(ctx, vzV1Beta1, updateOpts...); err != nil {
		return err
	}
	return nil
}

// UpdateV1Alpha1Status takes in a v1alpha1 Verrazzano struct and sends an update request to the K8s API server
// for that resource's status. This function internally converts the Verrazzano to v1beta1 before sending the update request.
func UpdateV1Alpha1Status(ctx context.Context, statusWriter client.StatusWriter, vzV1Alpha1 *v1alpha1.Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vzV1Alpha1.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	if err := statusWriter.Update(ctx, vzV1Beta1); err != nil {
		return err
	}
	return nil
}
