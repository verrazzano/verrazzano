// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
// Always for secrets and configmaps since they may contain module configuration
func (r Reconciler) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return []controllerspi.WatchDescriptor{
		{
			WatchedResourceKind: source.Kind{Type: &corev1.Secret{}},
			FuncShouldReconcile: r.ShouldSecretTriggerReconcile,
		},
		{
			WatchedResourceKind: source.Kind{Type: &corev1.ConfigMap{}},
			FuncShouldReconcile: r.ShouldConfigMapTriggerReconcile,
		},
	}
}

// ShouldSecretTriggerReconcile returns true if reconcile should be done in response to a Secret lifecycle event
func (r Reconciler) ShouldSecretTriggerReconcile(vzNSN types.NamespacedName, secret client.Object, _ controllerspi.WatchEvent) bool {
	if secret.GetNamespace() != vzNSN.Namespace {
		return false
	}
	vzcr := vzapi.Verrazzano{}
	if err := r.Client.Get(context.TODO(), vzNSN, &vzcr); err != nil {
		return false
	}
	secretNames := getOverrideResourceNames(&vzcr, secretType)
	_, ok := secretNames[secret.GetName()]
	return ok
}

// ShouldConfigMapTriggerReconcile returns true if reconcile should be done in response to a Secret lifecycle event
func (r Reconciler) ShouldConfigMapTriggerReconcile(vzNSN types.NamespacedName, cm client.Object, _ controllerspi.WatchEvent) bool {
	if cm.GetNamespace() != vzNSN.Namespace {
		return false
	}
	vzcr := vzapi.Verrazzano{}
	if err := r.Client.Get(context.TODO(), vzNSN, &vzcr); err != nil {
		return false
	}
	names := getOverrideResourceNames(&vzcr, configMapType)
	_, ok := names[cm.GetName()]
	return ok
}
