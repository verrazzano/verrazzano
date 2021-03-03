// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
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
	logger.Info("Fetching VerrazzanoProject")
	err := r.Get(ctx, req.NamespacedName, &vp)
	if err != nil {
		logger.Error(err, "Failed to fetch VerrazzanoProject")
		return result, client.IgnoreNotFound(err)
	}

	err = r.createOrUpdateNamespaces(ctx, vp)
	return result, err
}

func (r *Reconciler) createOrUpdateNamespaces(ctx context.Context, vp clustersv1alpha1.VerrazzanoProject) error {
	if vp.Namespace == constants.VerrazzanoMultiClusterNamespace {
		for _, nsTemplate := range vp.Spec.Template.Namespaces {
			r.Log.Info("create or update with underlying namespace", "namespace", nsTemplate.Metadata.Name)
			var namespace corev1.Namespace
			namespace.Name = nsTemplate.Metadata.Name

			opResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, &namespace, func() error {
				r.mutateNamespace(nsTemplate, &namespace)
				return nil
			})
			if err != nil {
				r.Log.Info("create or update namespace %s failed with result %q and error %v", nsTemplate.Metadata.Name, opResult, err)
			}
		}
	}
	return nil
}

func (r *Reconciler) mutateNamespace(nsTemplate clustersv1alpha1.NamespaceTemplate, namespace *corev1.Namespace) {
	namespace.Labels = nsTemplate.Metadata.Labels
	namespace.Annotations = nsTemplate.Metadata.Annotations
	namespace.Spec = nsTemplate.Spec
}
