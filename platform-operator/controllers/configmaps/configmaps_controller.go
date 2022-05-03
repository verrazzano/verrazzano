// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmaps

import (
	"context"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// VerrazzanoConfigMapsReconciler reconciles ConfigMaps.
// This controller manages Helm override sources from the Verrazzano CR
type VerrazzanoConfigMapsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    vzlog.VerrazzanoLogger
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoConfigMapsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *VerrazzanoConfigMapsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO List:
	// 1. Get the Verrazzano CR and verify that the Namespace of it and the request align
	//      a) i.e. vz.Namespace == req.Namespace
	// 2. Verify that the ConfigMap exists as a helm override (use vzconfig.vzContainsResources)
	// 3. Update the Verrazzano CR to start a helm upgrade command
	//      a) Instructions TBD
	// 4. Create unit tests for new functions

	return ctrl.Result{}, nil
}

func (r *VerrazzanoConfigMapsReconciler) initLogger(cm corev1.ConfigMap) (ctrl.Result, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cm.Name,
		Namespace:      cm.Namespace,
		ID:             string(cm.UID),
		Generation:     cm.Generation,
		ControllerName: "secrets",
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for VerrazzanoConfigMap controller", err)
		return newRequeueWithDelay(), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
