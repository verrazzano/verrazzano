// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"context"

	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles a VerrazzanoWebLogicWorkload object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoWebLogicWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoWebLogicWorkload resource. It fetches the embedded WebLogic CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("verrazzanoweblogicworkload", req.NamespacedName)
	log.Info("Reconciling verrazzano weblogic workload")

	return ctrl.Result{}, nil
}
