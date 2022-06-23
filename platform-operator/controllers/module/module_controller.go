// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package module

import (
	"context"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	modules2 "github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var delegates = map[string]func(*modulesv1alpha1.Module) modules2.DelegateReconciler{
	keycloak.ComponentName:  keycloak.NewComponent,
	coherence.ComponentName: coherence.NewComponent,
	weblogic.ComponentName:  weblogic.NewComponent,
}

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modulesv1alpha1.Module{}).
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
	module := &modulesv1alpha1.Module{}
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
		return ctrl.Result{}, nil
	}
	moduleCtx, err := spi.NewModuleContext(log, r.Client, &verrazzanos.Items[0], module, false)
	if err != nil {
		log.Errorf("Failed to create module context: %v", err)
		return clusters.NewRequeueWithDelay(), err
	}
	delegate.SetStatusWriter(r.Status())
	if err := delegate.ReconcileModule(moduleCtx); err != nil {
		return handleError(moduleCtx, err)
	}
	return ctrl.Result{}, nil
}

func getDelegateController(module *modulesv1alpha1.Module) modules2.DelegateReconciler {
	newDelegate := delegates[module.ObjectMeta.Labels[modules2.ControllerLabel]]
	if newDelegate == nil {
		return nil
	}
	return newDelegate(module)
}

func handleError(ctx spi.ComponentContext, err error) (ctrl.Result, error) {
	log := ctx.Log()
	module := ctx.Module()
	if k8serrors.IsConflict(err) {
		log.Debugf("Conflict resolving module %s", module.Name)
	} else if modules2.IsNotReadyError(err) {
		log.Progressf("Module %s is not ready yet", module.Name)
	} else {
		log.Errorf("Failed to reconcile module %s/%s: %v", module.Name, module.Namespace, err)
		return clusters.NewRequeueWithDelay(), err
	}
	return clusters.NewRequeueWithDelay(), nil
}
