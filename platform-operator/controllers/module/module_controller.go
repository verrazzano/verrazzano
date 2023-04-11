// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package module

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	controller2 "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	modules2 "github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/reconciler"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

var delegates = map[string]func(*modulesv1beta2.ModuleLifecycle) modules2.DelegateReconciler{
	//keycloak.ComponentName:  keycloak.NewComponent,
	//coherence.ComponentName: coherence.NewComponent,
	//weblogic.ComponentName:  weblogic.NewComponent,
}

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modulesv1beta2.ModuleLifecycle{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	verrazzanos := &vzapi.VerrazzanoList{}
	if err := r.List(ctx, verrazzanos); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get Verrazzanos %s/%s", req.Namespace, req.Name)
		return clusters.NewRequeueWithDelay(), err
	}

	// Get the module for the request
	module := &modulesv1beta2.ModuleLifecycle{}
	if err := r.Get(ctx, req.NamespacedName, module); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get Module %s/%s", req.Namespace, req.Name)
		return clusters.NewRequeueWithDelay(), err
	}
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           module.Name,
		Namespace:      module.Namespace,
		ID:             string(module.UID),
		Generation:     module.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Module controller: %v", err)
		return clusters.NewRequeueWithDelay(), err
	}

	if module.Generation == module.Status.ObservedGeneration {
		log.Debugf("Skipping module %s reconcile, observed generation has not change", module.Name)
		return ctrl.Result{}, nil
	}

	// Unknown module controller cannot be handled
	delegate := getDelegateController(module)
	if delegate == nil {
		return ctrl.Result{}, fmt.Errorf("No delegate found for module %s/%s", module.Namespace, module.Name)
	}
	moduleCtx, err := r.createComponentContext(log, verrazzanos, module)
	if err != nil {
		return controller2.NewRequeueWithDelay(2, 5, time.Second), err
	}

	delegate.SetStatusWriter(r.Status())
	if err := delegate.ReconcileModule(moduleCtx); err != nil {
		return handleError(moduleCtx, err)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) createComponentContext(log vzlog.VerrazzanoLogger, verrazzanos *vzapi.VerrazzanoList, module *modulesv1beta2.ModuleLifecycle) (spi.ComponentContext, error) {
	var moduleCtx spi.ComponentContext
	var err error
	//if len(verrazzanos.Items) > 0 {
	//	moduleCtx, err = spi.NewModuleContext(log, r.Client, &verrazzanos.Items[0], module, false)
	//} else {
	//	moduleCtx, err = spi.NewMinimalModuleContext(r.Client, log, module, false)
	//}
	//if err != nil {
	//	log.Errorf("Failed to create module context: %v", err)
	//}
	return moduleCtx, err
}

func getDelegateController(module *modulesv1beta2.ModuleLifecycle) modules2.DelegateReconciler {
	newDelegate := delegates[module.ObjectMeta.Labels[modules2.ControllerLabel]]
	if newDelegate == nil {
		// If an existing delegate does not exist, wrap it in a Helm adapter to just do helm stuff
		return reconciler.NewHelmAdapter(module)
	}
	return newDelegate(module)
}

func handleError(ctx spi.ComponentContext, err error) (ctrl.Result, error) {
	//log := ctx.Log()
	//module := ctx.Module()
	//if k8serrors.IsConflict(err) {
	//	log.Debugf("Conflict resolving module %s", module.Name)
	//} else if modules2.IsNotReadyError(err) {
	//	log.Progressf("Module %s is not ready yet", module.Name)
	//} else {
	//	log.Errorf("Failed to reconcile module %s/%s: %v", module.Name, module.Namespace, err)
	//	return clusters.NewRequeueWithDelay(), err
	//}
	//return clusters.NewRequeueWithDelay(), nil
	return ctrl.Result{}, nil
}
