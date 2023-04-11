// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package modlifecycle

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/common"
	"time"

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
)

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

	// Get the mlc for the request
	mlc := &modulesv1beta2.ModuleLifecycle{}
	if err := r.Get(ctx, req.NamespacedName, mlc); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		zap.S().Errorf("Failed to get Module %s/%s", req.Namespace, req.Name)
		return clusters.NewRequeueWithDelay(), err
	}
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           mlc.Name,
		Namespace:      mlc.Namespace,
		ID:             string(mlc.UID),
		Generation:     mlc.Generation,
		ControllerName: "mlc-lifecycle",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for ModuleLifecycle controller: %v", err)
		return clusters.NewRequeueWithDelay(), err
	}

	// TODO: in reality this may perhaps be broken out into separate operators/controllers, but then the lifecycle
	//   CRs would need to be potentially extensible, or the operators/controllers would distinguish ownership of a
	//   ModuleLifecycle instance by field/labels/annotations, which seems likeß a controller anti-pattern
	delegate, err := reconciler.New(mlc, r.Status())
	if err != nil {
		// Unknown mlc controller cannot be handled; no need to re-reconcile until the resource is updated
		log.Errorf("Error retrieving delegate lifecycle reconciler: %v", err.Error())
		return ctrl.Result{}, err
	}

	if delegate == nil {
		return ctrl.Result{}, fmt.Errorf("No delegate found for mlc %s/%s", mlc.Namespace, mlc.Name)
	}
	if err != nil {
		return controller2.NewRequeueWithDelay(2, 5, time.Second), err
	}

	result, err := delegate.Reconcile(log, r.Client, mlc)
	if err != nil {
		return handleError(log, mlc, err)
	}
	return result, nil
}

func handleError(log vzlog.VerrazzanoLogger, mlc *modulesv1beta2.ModuleLifecycle, err error) (ctrl.Result, error) {
	if k8serrors.IsConflict(err) {
		log.Debugf("Conflict resolving module lifecycle %s", common.GetNamespacedName(mlc.ObjectMeta))
	} else if delegates.IsNotReadyError(err) {
		log.Progressf("Module lifecycle %s is not ready yet", common.GetNamespacedName(mlc.ObjectMeta))
	} else {
		log.Errorf("failed to reconcile module %s: %v", common.GetNamespacedName(mlc.ObjectMeta), err)
		return clusters.NewRequeueWithDelay(), err
	}
	return clusters.NewRequeueWithDelay(), nil
}
