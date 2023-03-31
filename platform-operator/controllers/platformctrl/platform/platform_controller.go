// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package platform

import (
	"context"
	"time"

	vzcontroller "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	installv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PlatformReconciler reconciles a Verrazzano Platform object
type PlatformReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

// Name of finalizer
const finalizerName = "platform.verrazzano.io"

// SetupWithManager creates a new controller and adds it to the manager
func (r *PlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&installv1beta2.Platform{}).Build(r)
	return err
}

// Reconcile the Verrazzano CR
// +kubebuilder:rbac:groups=platform.verrazzano.io,resources=platforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.verrazzano.io,resources=platforms/status,verbs=get;update;patch
func (r *PlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO: Metrics setup

	platformInstance := &installv1beta2.Platform{}
	if err := r.Get(ctx, req.NamespacedName, platformInstance); err != nil {
		// TODO: errorCounterMetricObject.Inc()
		// If the resource is not found, that means all the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Platform resource: %v", err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           platformInstance.Name,
		Namespace:      platformInstance.Namespace,
		ID:             string(platformInstance.UID),
		Generation:     platformInstance.Generation,
		ControllerName: "platform",
	})
	if err != nil {
		// TODO: errorCounterMetricObject.Inc()
		zap.S().Errorf("Failed to create controller logger for Platform controller: %v", err)
	}

	log.Infof("Reconciling platform instance %s/%s", platformInstance.Namespace, platformInstance.Name)

	// Check if resource is being deleted
	if !platformInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Oncef("Removing finalizer %s", finalizerName)
		platformInstance.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(platformInstance.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, platformInstance); err != nil {
			return newRequeueWithDelay(), err
		}
		return ctrl.Result{}, nil
	}

	if !vzstring.SliceContainsString(platformInstance.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		platformInstance.ObjectMeta.Finalizers = append(platformInstance.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(context.TODO(), platformInstance); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	// Update the platform status
	platformInstance.Status.Version = platformInstance.Spec.Version
	if err := r.Status().Update(context.TODO(), platformInstance); err != nil {
		return newRequeueWithDelay(), err
	}

	if err := r.doReconcile(log, platformInstance); err != nil {
		return newRequeueWithDelay(), err
	}

	log.Infof("Reconcile of platform instance %s/%s complete", platformInstance.Namespace, platformInstance.Name)
	return ctrl.Result{}, nil
}

func (r *PlatformReconciler) doReconcile(_ vzlog.VerrazzanoLogger, _ *installv1beta2.Platform) error {
	return nil
}

func newRequeueWithDelay() ctrl.Result {
	return vzcontroller.NewRequeueWithDelay(2, 5, time.Second)
}
