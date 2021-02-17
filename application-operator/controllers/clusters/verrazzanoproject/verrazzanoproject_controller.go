// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

// Reconciler reconciles a VerrazzanoProject object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoProject{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoProject resource.
// It fetches its namespaces if the VerrazzanoProject is in the verrazzano-mc namespace
// and create namespaces in the local cluster.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("verrazzanoproject", req.NamespacedName)
	var vp clustersv1alpha1.VerrazzanoProject
	result := reconcile.Result{}
	ctx := context.Background()
	logger.Info(" Fetching VerrazzanoProject", req.NamespacedName)
	err := r.Get(ctx, req.NamespacedName, &vp)
	if err != nil {
		logger.Info("Failed to fetch VerrazzanoProject", req.NamespacedName, "err", err)
		return result, client.IgnoreNotFound(err)
	}

	err = r.createOrUpdateNamespaces(ctx, vp, logger)
	return result, err
}

func (r *Reconciler) createOrUpdateNamespaces(ctx context.Context, vp clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	if vp.Namespace == constants.VerrazzanoMultiClusterNamespace {
		for _, namespace := range vp.Spec.Namespaces {
			logger.Info("VerrazzanoProject", vp.Name, "create or update with underlying namespace", namespace)
			nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			controllerutil.CreateOrUpdate(ctx, r.Client, nsSpec, func() error {
				return nil
			})
		}
	}
	return nil
}
