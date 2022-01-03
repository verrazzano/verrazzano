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

// Reconciler reconciles a metrics workload object
type Reconciler struct {
	k8sclient.Client
	Log     logr.Logger
	Scheme  *k8sruntime.Scheme
	Scraper string
}

// setupWithManagerForGVK creates a controller for a specific GKV and adds it to the manager.
func (r *Reconciler) setupWithManagerForGVK(mgr k8scontroller.Manager, group string, version string, kind string) error {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	return k8scontroller.NewControllerManagedBy(mgr).For(&u).Complete(r)
}

// SetupWithManager creates controllers for each supported GKV and adds it to the manager
// See https://book-v1.book.kubebuilder.io/beyond_basics/controller_watches.html for potentially better way to watch arbitrary resources
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	//TODO: Need some way to lookup set of supported metwork workload GKVs.
	if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "Deployment"); err != nil {
		return err
	}
	// Disabling for now as Domain and Coherence cause problems when those CRDs don't exist.
	//if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "ReplicaSet"); err != nil {
	//	return err
	//}
	//if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "StatefulSet"); err != nil {
	//	return err
	//}
	//if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "DaemonSet"); err != nil {
	//	return err
	//}
	//if err := r.setupWithManagerForGVK(mgr, "weblogic.oracle", "v7", "Domain"); err != nil {
	//	return err
	//}
	//if err := r.setupWithManagerForGVK(mgr, "weblogic.oracle", "v8", "Domain"); err != nil {
	//	return err
	//}
	//if err := r.setupWithManagerForGVK(mgr, "coherence.oracle.com", "v1", "Coherence"); err != nil {
	//	return err
	//}
	return nil
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application operator is now manually managed.
func (r *Reconciler) Reconcile(req k8scontroller.Request) (k8scontroller.Result, error) {
	r.Log.V(1).Info("Reconcile metrics scrape config", "resource", req.NamespacedName)
	//TODO: To be implemented
	return k8sreconcile.Result{}, nil
}
