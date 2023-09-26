// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/controller/custom"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
// Always watch for secrets and configmaps since they may contain module configuration
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
		{
			WatchedResourceKind: source.Kind{Type: &batchv1.Job{}},
			FuncShouldReconcile: r.ShouldJobTriggerReconcile,
		},
	}
}

// ShouldSecretTriggerReconcile returns true if reconcile should be done in response to a Secret lifecycle event
func (r Reconciler) ShouldSecretTriggerReconcile(cli client.Client, wev controllerspi.WatchEvent) bool {
	if wev.NewWatchedObject.GetNamespace() != wev.ReconcilingResource.Namespace {
		return false
	}
	vzcr := vzv1alpha1.Verrazzano{}
	if err := r.Client.Get(context.TODO(), wev.ReconcilingResource, &vzcr); err != nil {
		return false
	}
	secretNames := getOverrideResourceNames(&vzcr, secretType)
	_, ok := secretNames[wev.NewWatchedObject.GetName()]
	return ok
}

// ShouldConfigMapTriggerReconcile returns true if reconcile should be done in response to a ConfigMap lifecycle event
func (r Reconciler) ShouldConfigMapTriggerReconcile(cli client.Client, wev controllerspi.WatchEvent) bool {
	if wev.NewWatchedObject.GetNamespace() != wev.ReconcilingResource.Namespace {
		return false
	}
	vzcr := vzv1alpha1.Verrazzano{}
	if err := r.Client.Get(context.TODO(), wev.ReconcilingResource, &vzcr); err != nil {
		return false
	}
	names := getOverrideResourceNames(&vzcr, configMapType)
	_, ok := names[wev.NewWatchedObject.GetName()]
	return ok
}

// ShouldJobTriggerReconcile returns true if reconcile should be done in response to a Job lifecycle event
func (r Reconciler) ShouldJobTriggerReconcile(cli client.Client, wev controllerspi.WatchEvent) bool {
	return custom.IsMysqlOperatorJob(cli, wev.NewWatchedObject)
}
