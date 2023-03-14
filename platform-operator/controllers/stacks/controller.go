// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stacks

// TODO COPIED FROM module-poc module controller https://raw.githubusercontent.com/verrazzano/verrazzano/mcico/module-poc/platform-operator/controllers/module/module_controller.go
// TODO Adapt for stack purposes.
import (
	"context"
	"fmt"
	"reflect"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/stacks/monitoring"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/stacks/stackspi"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

/*var delegates = map[string]func(*modulesv1alpha1.Module) modules2.DelegateReconciler{
	keycloak.ComponentName:  keycloak.NewComponent,
	coherence.ComponentName: coherence.NewComponent,
	weblogic.ComponentName:  weblogic.NewComponent,
}*/

var stackComponents = map[string]stackspi.StackComponent{}

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	initStackComponentList()
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(r.createStackConfigMapPredicate()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

func initStackComponentList() {
	stackComponents[monitoring.StackName] = monitoring.NewStackComponent()
}

func (r *Reconciler) createStackConfigMapPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.isVerrazzanoStackConfigMap(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.isVerrazzanoStackConfigMap(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.isVerrazzanoStackConfigMap(e.ObjectNew)
		},
	}
}

func (r *Reconciler) isVerrazzanoStackConfigMap(o client.Object) bool {
	configMap := o.(*v1.ConfigMap)
	if stackName := configMap.Annotations[vzconst.VerrazzanoStackAnnotationName]; stackName == "" {
		return false
	}
	return true
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
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	zap.S().Infof("DEVA Reconciling Stack for configmap %s/%s", req.Namespace, req.Name)
	// Get the stack configmap for the request
	stackConfig := v1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, &stackConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get Stack ConfigMap %s/%s", req.Namespace, req.Name)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	stackName := stackConfig.Annotations[vzconst.VerrazzanoStackAnnotationName]
	if stackName == "" {
		err := fmt.Errorf("Stack ConfigMap reconcile called %s/%s, but does not have stack annotation %s",
			req.Namespace, req.Name, vzconst.VerrazzanoStackAnnotationName)
		zap.S().Errorf(err.Error())
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	zap.S().Infof("DEVA Reconcile retrieved configmap for stack %s", stackName)

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           stackName,
		Namespace:      stackConfig.Namespace,
		ID:             string(stackConfig.UID),
		Generation:     stackConfig.Generation,
		ControllerName: "verrazzanostack",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Stack controller: %v", err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	zap.S().Infof("DEVA Created logger for stack %s - here's a message", stackName)
	log.Infof("DEVA msg from stack logger")

	// TODO kick off install of stack component indicated by the ConfigMap
	stackComponent, found := stackComponents[stackName]
	if !found {
		log.Errorf("Failed to find stack component with name %s", stackName)
		return newRequeueWithDelay(), nil
	}
	log.Infof("DEVA found stack component %v", stackComponent)
	vzlist := &vzapi.VerrazzanoList{}
	var vz vzapi.Verrazzano
	if err := r.List(ctx, vzlist); err != nil {
		// If the resource is not found or the list is empty, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Errorf("Failed to list Verrazzano resources: %v", err)
		return newRequeueWithDelay(), nil
	}
	if len(vzlist.Items) > 0 {
		vz = vzlist.Items[0]
	} else {
		log.Errorf("No Verrazzano resource found. Nothing to do.")
		return reconcile.Result{}, nil
	}
	log.Infof("DEVA fetched Verrazzano")
	if reflect.TypeOf(stackComponent).AssignableTo(reflect.TypeOf((*stackspi.StackComponent)(nil)).Elem()) {
		// Keep retrying to reconcile components until it completes
		// vzctx, err := vzcontext.NewVerrazzanoContext(log, r.Client, vz, r.DryRun)
		stackCtx, err := stackspi.NewStackContext(log, r.Client, &vz, nil, stackConfig, r.DryRun)
		if err != nil {
			zap.S().Errorf("Failed to create Stack Context: %v", err)
			return newRequeueWithDelay(), err
		}
		if err := stackComponent.(stackspi.StackComponent).ReconcileStack(stackCtx); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	// return ctrl.Result{}, nil
	/*if stackConfig.Generation == stackConfig.Status.ObservedGeneration {
		log.Debugf("Skipping stack %s reconcile, observed generation has not changed for ConfigMap %s/%s",
			stackName, stackConfig.Namespace, stackConfig.Name)
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
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	delegate.SetStatusWriter(r.Status())
	if err := delegate.ReconcileModule(moduleCtx); err != nil {
		return handleError(moduleCtx, err)
	}*/
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconciliation requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}

/*func getDelegateController(module *modulesv1alpha1.Module) modules2.DelegateReconciler {
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
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second), nil
}*/
