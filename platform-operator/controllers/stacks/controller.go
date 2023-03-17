// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stacks

// TODO COPIED FROM module-poc module controller https://raw.githubusercontent.com/verrazzano/verrazzano/mcico/module-poc/platform-operator/controllers/module/module_controller.go
// TODO Adapt for stack purposes.
import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
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

const (
	componentNameKey      = "name"
	componentNamespaceKey = "namespace"
	chartURLKey           = "chartURL"
	overridesKey          = "overrides"
)

type Reconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(r.createStackConfigMapPredicate()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
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
		if k8serrors.IsNotFound(err) || len(verrazzanos.Items) == 0 {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get Verrazzanos %s/%s", req.Namespace, req.Name)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	vz := verrazzanos.Items[0]

	zap.S().Infof("DEVA Reconciling Stack for configmap %s/%s", req.Namespace, req.Name)
	// Get the configmap for the request
	cm := v1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get ConfigMap %s/%s", req.Namespace, req.Name)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	stackName := cm.Annotations[vzconst.VerrazzanoStackAnnotationName]
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
		Namespace:      cm.Namespace,
		ID:             string(cm.UID),
		Generation:     cm.Generation,
		ControllerName: "verrazzanostack",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Stack controller: %v", err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	zap.S().Infof("DEVA Created logger for stack %s - here's a message", stackName)
	log.Infof("DEVA msg from stack logger")

	comp, err := newDevComponent(log, cm)
	if err != nil {
		log.Errorf("Failed to read component %s data from configMap %s: %v", stackName, cm.GetName(), err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}

	compCtx, err := spi.NewContext(log, r.Client, &vz, nil, false)
	if err != nil {
		log.Errorf("Failed to create context: %v", err)
	}
	err = comp.Install(compCtx)
	if err != nil {
		log.Errorf("Failed to install component %s from configMap %s: ", comp.ReleaseName, cm.GetName(), err)
		return vzctrl.NewRequeueWithDelay(2, 3, time.Second), err
	}
	return ctrl.Result{}, nil
}
