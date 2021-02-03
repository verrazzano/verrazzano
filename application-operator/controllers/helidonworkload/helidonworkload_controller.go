// Copyright (c) 2021 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"context"

	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a VerrazzanoHelidonWorkload object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoHelidonWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoHelidonWorkload resource. It fetches the embedded Deployment CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanohelidonworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanohelidonworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanohelidonworkload", req.NamespacedName)
	log.Info("Reconciling verrazzano helidon workload")

	// fetch the workload and unwrap the Deployment resource
	_, err := r.fetchWorkload(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Successfully created Verrazzano Helidon workload")
	return ctrl.Result{}, nil
}

// fetchWorkload fetches the VerrazzanoHelidonWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName) (*vzapi.VerrazzanoHelidonWorkload, error) {
	var workload vzapi.VerrazzanoHelidonWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("VerrazzanoHelidonWorkload has been deleted", "name", name)
		} else {
			r.Log.Error(err, "Failed to fetch VerrazzanoHelidonWorkload", "name", name)
		}
		return nil, err
	}

	return &workload, nil
}
