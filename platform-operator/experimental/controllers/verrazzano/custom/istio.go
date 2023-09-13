// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
)

// DeleteIstioCARootCert deletes the Istio root cert ConfigMap that gets distributed across the cluster
func DeleteIstioCARootCert(ctx componentspi.ComponentContext) error {
	namespaces := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &namespaces)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the cluster namespaces: %v", err)
	}

	for _, ns := range namespaces.Items {
		err := resource.Resource{
			Name:      istioRootCertName,
			Namespace: ns.GetName(),
			Client:    ctx.Client(),
			Object:    &corev1.ConfigMap{},
			Log:       ctx.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}
