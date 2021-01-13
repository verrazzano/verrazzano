// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oamv1alpha1 "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"

	vzapi "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
)

// Reconciler reconciles a LoggingScope object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingscopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingscopes/status,verbs=get;update;patch

// Reconcile reconciles a LoggingScope.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("scope", req.NamespacedName)
	log.Info("Reconcile logging scope")

	// Fetch the scope.
	scope, err := r.fetchScope(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if scope != nil {
		fmt.Printf("%s", scope.Name)
	}

	return ctrl.Result{}, nil
}

// fetchScope attempts to get a scope given a namespaced name.
// Will return nil for the scope and no error if the scope does not exist.
func (r *Reconciler) fetchScope(ctx context.Context, name types.NamespacedName) (*vzapi.LoggingScope, error) {
	var scope vzapi.LoggingScope
	r.Log.Info("Fetch scope", "name", name)
	if err := r.Get(ctx, name, &scope); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("Scope has been deleted")
			return nil, nil
		}
		r.Log.Info("Failed to fetch scope")
		return nil, err
	}
	return &scope, nil
}

// SetupWithManager creates a controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1alpha1.LoggingScope{}).
		Complete(r)
}
