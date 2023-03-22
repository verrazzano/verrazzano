// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package components

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ComponentConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	DryRun bool
}

func (r *ComponentConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(r.createComponentConfigMapPredicate()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

func (r *ComponentConfigMapReconciler) createComponentConfigMapPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.isComponentConfigMap(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.isComponentConfigMap(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.isComponentConfigMap(e.ObjectNew)
		},
	}
}

func (r *ComponentConfigMapReconciler) isComponentConfigMap(o client.Object) bool {
	configMap := o.(*v1.ConfigMap)
	return configMap.Labels[devComponentConfigMapAPIVersionLabel] == devComponentConfigMapAPIVersionv1beta2 &&
		configMap.Labels[devComponentConfigMapKindLabel] != ""
}

// Reconcile function for the ComponentConfigMapReconciler
func (r *ComponentConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	verrazzanos := &vzapi.VerrazzanoList{}
	err := r.List(ctx, verrazzanos)
	if err != nil && !k8serrors.IsNotFound(err) {
		zap.S().Errorf("Failed to get Verrazzanos %s/%s", req.Namespace, req.Name)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	if err != nil || len(verrazzanos.Items) == 0 {
		zap.S().Debug("No Verrazzanos found in the cluster")
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	vz := verrazzanos.Items[0]

	zap.S().Infof("Reconciling component configmap %s/%s", req.Namespace, req.Name)
	// Get the configmap for the request
	cm := v1.ConfigMap{}
	if err = r.Get(ctx, req.NamespacedName, &cm); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get configmap %s/%s", req.Namespace, req.Name)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	if cm.Namespace != vz.Namespace {
		err = fmt.Errorf("Component ConfigMap must be in the same namespace as the Verrazzano resource, ConfigMap namespace: %s, Verrazzano namespace: %s", cm.Namespace, vz.Namespace)
		zap.S().Error(err)
		return ctrl.Result{}, err
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cm.Name,
		Namespace:      cm.Namespace,
		ID:             string(cm.UID),
		Generation:     cm.Generation,
		ControllerName: "verrazzanodevcomponent",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for component configmap %s/%s: %v", cm.GetName(), cm.GetNamespace(), err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	if cm.Labels[devComponentConfigMapKindLabel] != devComponentConfigMapKindHelmComponent {
		err = fmt.Errorf("%s is not a support configmap-kind, %s is the only configmap-kind supported",
			cm.Labels[devComponentConfigMapKindLabel], devComponentConfigMapKindHelmComponent)
		log.Error(err)
		return ctrl.Result{}, err
	}

	comp, err := newDevHelmComponent(cm)
	if err != nil {
		log.Errorf("Failed to read component %s data from configmap %s/%s: %v", cm.GetName(), cm.GetNamespace(), err)
		// don't requeue if the data is invalid
		// once the data is updated to be correct it will trigger another reconcile
		return ctrl.Result{}, err
	}

	compCtx, err := spi.NewContext(log, r.Client, &vz, nil, false)
	if err != nil {
		log.Errorf("Failed to create context: %v", err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	// install dev component
	// TODO: turn this into a state machine
	err = comp.Install(compCtx)
	if err != nil {
		log.Errorf("Failed to install component %s from configMap %s: ", comp.ReleaseName, cm.GetName(), err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	log.Infof("Successfully installed component %s", comp.ReleaseName)
	return ctrl.Result{}, nil
}
