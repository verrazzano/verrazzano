// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a MetricsTrait object
type Reconciler struct {
	k8sclient.Client
	Log     logr.Logger
	Scheme  *k8sruntime.Scheme
	Scraper string
}

// SetupWithManager creates a controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	//TODO: See https://book-v1.book.kubebuilder.io/beyond_basics/controller_watches.html for watching arbitrary resources
	//TODO: Need some way to lookup a registered set of workload GKVs
	types := []*unstructured.Unstructured{
		newUnstructured("apps", "v1", "Deployment"),
		newUnstructured("apps", "v1", "ReplicaSet"),
		newUnstructured("apps", "v1", "StatefulSet"),
		newUnstructured("apps", "v1", "DaemonSet"),
		newUnstructured("weblogic.oracle", "v7", "Domain"),
		newUnstructured("weblogic.oracle", "v8", "Domain"),
		newUnstructured("coherence.oracle.com", "v1", "Coherence"),
	}
	c := k8scontroller.NewControllerManagedBy(mgr)
	for _, t := range types {
		c = c.For(t)
	}
	return c.Complete(r)
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application operator is now manually managed.
func (r *Reconciler) Reconcile(req k8scontroller.Request) (k8scontroller.Result, error) {
	r.Log.V(1).Info("Reconcile metrics scrape config", "resource", req.NamespacedName)
	//TODO: To be implemented
	//
	return k8sreconcile.Result{}, nil
}

func newUnstructured(group string, version string, kind string) *unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})
	return &u
}
