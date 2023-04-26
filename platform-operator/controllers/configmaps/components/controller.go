// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package components

import (
	"context"
	"fmt"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var shimComponents = map[string]spi.Component{}

type ComponentConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	DryRun bool
}

func initShimComponentList() {
	// Add any shim components here that you want to test.
	// For example,
	// shimComponents[capi.ComponentName] = capi.NewComponent()
}

func (r *ComponentConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	initShimComponentList()
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
		return newRequeueWithDelay(), err
	}
	if err != nil || len(verrazzanos.Items) == 0 {
		zap.S().Debug("No Verrazzanos found in the cluster")
		return newRequeueWithDelay(), err
	}
	vz := verrazzanos.Items[0]

	zap.S().Infof("Reconciling component configmap %s/%s", req.Namespace, req.Name)
	// Get the configmap for the request
	configMap := &v1.ConfigMap{}
	if err = r.Get(ctx, req.NamespacedName, configMap); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get configmap %s/%s", req.Namespace, req.Name)
		return newRequeueWithDelay(), err
	}

	if configMap.Namespace != vz.Namespace {
		err = fmt.Errorf("Component ConfigMap must be in the same namespace as the Verrazzano resource, ConfigMap namespace: %s, Verrazzano namespace: %s", configMap.Namespace, vz.Namespace)
		zap.S().Error(err)
		return ctrl.Result{}, err
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           configMap.Name,
		Namespace:      configMap.Namespace,
		ID:             string(configMap.UID),
		Generation:     configMap.Generation,
		ControllerName: "verrazzanodevcomponent",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for component configmap %s/%s: %v", configMap.GetName(), configMap.GetNamespace(), err)
		return newRequeueWithDelay(), err
	}

	var comp spi.Component
	if configMap.Labels[devComponentConfigMapKindLabel] == devComponentConfigMapKindHelmComponent {
		comp, err = newDevHelmComponent(configMap)
	} else if configMap.Labels[devComponentConfigMapKindLabel] == devComponentConfigMapKindShimComponent {
		comp, err = newDevShimComponent(configMap)
	} else {
		err = fmt.Errorf("%s is not a support configmap-kind, %s and %s are the only configmap-kind supported",
			configMap.Labels[devComponentConfigMapKindLabel], devComponentConfigMapKindHelmComponent, devComponentConfigMapKindShimComponent)
	}

	if err != nil {
		log.Errorf("Invalid configmap %s/%s: %v", configMap.GetNamespace(), configMap.GetName(), err)
		return ctrl.Result{}, err
	}

	compCtx, err := spi.NewContext(log, r.Client, &vz, nil, false)
	if err != nil {
		log.Errorf("Failed to create context: %v", err)
		return newRequeueWithDelay(), err
	}

	return r.processComponent(compCtx, comp, configMap)
}

func (r *ComponentConfigMapReconciler) processComponent(ctx spi.ComponentContext, comp spi.Component, configMap *v1.ConfigMap) (ctrl.Result, error) {
	// check if component is being deleted
	if !configMap.DeletionTimestamp.IsZero() {
		// uninstall component
		if err := doUninstall(ctx, comp); err != nil {
			ctx.Log().Errorf("Error uninstalling dev component %s: %v", comp.Name(), err)
			return newRequeueWithDelay(), err
		}
		ctx.Log().Infof("Successfully uninstalled dev component %s", comp.Name())

		// remove finalizer to delete the component
		controllerutil.RemoveFinalizer(configMap, constants.DevComponentFinalizer)
		err := r.Update(context.TODO(), configMap)
		if err != nil {
			ctx.Log().Errorf("Error removing finalizer %s for dev component %s: %v", constants.DevComponentFinalizer, comp.Name(), err)
			return newRequeueWithDelay(), err
		}
		ctx.Log().Infof("dev component %s has been successfully uninstalled", comp.Name())
		return reconcile.Result{Requeue: true}, nil
	}

	// Check if our finalizer is already present and add it if not
	if !controllerutil.ContainsFinalizer(configMap, constants.DevComponentFinalizer) {
		configMap.Finalizers = append(configMap.Finalizers, constants.DevComponentFinalizer)
		err := r.Update(context.TODO(), configMap)
		if err != nil {
			ctx.Log().Errorf("Error adding finalizer %s for dev component %s: %v", constants.DevComponentFinalizer, comp.Name(), err)
			return newRequeueWithDelay(), err
		}
		ctx.Log().Infof("Successfully added finalizer %s to configmap %s for dev component %s", constants.DevComponentFinalizer, configMap.Name, comp.Name())
		// adding finalizer to ConfigMap will trigger a requeue so no need to requeue here
		return reconcile.Result{}, nil
	}

	// if the release has not been installed, install it
	installed, err := comp.IsInstalled(ctx)
	if err != nil {
		ctx.Log().Errorf("Error checking release status for release %s: %v", comp.Name(), err)
		return newRequeueWithDelay(), err
	}
	if !installed {
		if err = doInstall(ctx, comp); err != nil {
			ctx.Log().Errorf("Error installing dev component %s: %v", comp.Name(), err)
			return newRequeueWithDelay(), err
		}
		ctx.Log().Infof("dev component %s has been successfully installed", comp.Name())
		return reconcile.Result{}, nil
	}

	// if the release has already been installed, upgrade it
	if err = doUpgrade(ctx, comp); err != nil {
		ctx.Log().Errorf("Error upgrading dev component %s: %v", comp.Name(), err)
		return newRequeueWithDelay(), err
	}
	ctx.Log().Infof("dev component %s has been successfully upgraded", comp.Name())
	return reconcile.Result{}, nil

}

// Create a new Result that will cause reconcile to requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}

func doInstall(ctx spi.ComponentContext, comp spi.Component) error {
	if err := comp.PreInstall(ctx); err != nil {
		return err
	}
	if err := comp.Install(ctx); err != nil {
		return err
	}
	for {
		if !comp.IsReady(ctx) {
			ctx.Log().Progressf("Component %s has been installed. Waiting for the component to be ready", comp.Name())
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}
	return comp.PostInstall(ctx)
}

func doUpgrade(ctx spi.ComponentContext, comp spi.Component) error {
	if err := comp.PreUpgrade(ctx); err != nil {
		return err
	}
	if err := comp.Upgrade(ctx); err != nil {
		return err
	}
	return comp.PostUpgrade(ctx)
}

func doUninstall(ctx spi.ComponentContext, comp spi.Component) error {
	if err := comp.PreUninstall(ctx); err != nil {
		return err
	}
	if err := comp.Uninstall(ctx); err != nil {
		return err
	}
	return comp.PostUninstall(ctx)
}
