// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package modlifecycle

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	controller2 "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/modlifecycle/delegates"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/modlifecycle/reconciler"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

var delegateReconcilersMap = map[string]func(*modulesv1beta2.ModuleLifecycle) delegates.DelegateReconciler{
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

	//verrazzanos := &vzapi.VerrazzanoList{}
	//if err := r.List(ctx, verrazzanos); err != nil {
	//	if k8serrors.IsNotFound(err) {
	//		return ctrl.Result{}, nil
	//	}
	//	zap.S().Errorf("Failed to get Verrazzanos %s/%s", req.Namespace, req.Name)
	//	return clusters.NewRequeueWithDelay(), err
	//}

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
		ControllerName: "module-lifecycle",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for ModuleLifecycle controller: %v", err)
		return clusters.NewRequeueWithDelay(), err
	}

	if module.Generation == module.Status.ObservedGeneration {
		log.Debugf("Skipping module %s reconcile, observed generation has not change", module.Name)
		return ctrl.Result{}, nil
	}

	// TODO: in reality this may perhaps be broken out into separate operators/controllers, but then the lifecycle
	//   CRs would need to be potentially extensible, or the operators/controllers would distinguish ownership of a
	//   ModuleLifecycle instance by field/labels/annotations, which seems likeß a controller anti-pattern
	// Unknown module controller cannot be handled
	delegate := getDelegateLifecycleController(module)
	if delegate == nil {
		return ctrl.Result{}, fmt.Errorf("No delegate found for module %s/%s", module.Namespace, module.Name)
	}
	// TODO: Shim layer
	//moduleCtx, err := r.createComponentContext(log, verrazzanos, module)
	if err != nil {
		return controller2.NewRequeueWithDelay(2, 5, time.Second), err
	}

	delegate.SetStatusWriter(r.Status())
	if err := delegate.ReconcileModule(log, r.Client, module); err != nil {
		return handleError(log, module, err)
	}
	return ctrl.Result{}, nil
}

func getDelegateLifecycleController(module *modulesv1beta2.ModuleLifecycle) delegates.DelegateReconciler {
	newDelegate := delegateReconcilersMap[module.ObjectMeta.Labels[delegates.ControllerLabel]]
	if newDelegate == nil {
		// If an existing delegate does not exist, wrap it in a Helm adapter to just do helm stuff
		return reconciler.NewHelmAdapter(module)
	}
	return newDelegate(module)
}

func handleError(log vzlog.VerrazzanoLogger, mlc *modulesv1beta2.ModuleLifecycle, err error) (ctrl.Result, error) {
	if k8serrors.IsConflict(err) {
		log.Debugf("Conflict resolving module %s", mlc.Name)
	} else if delegates.IsNotReadyError(err) {
		log.Progressf("Module %s is not ready yet", mlc.Name)
	} else {
		log.Errorf("Failed to reconcile module %s/%s: %v", mlc.Name, mlc.Namespace, err)
		return clusters.NewRequeueWithDelay(), err
	}
	return clusters.NewRequeueWithDelay(), nil
}
