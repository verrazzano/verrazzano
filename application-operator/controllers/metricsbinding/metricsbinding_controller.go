// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a metrics workload object
type Reconciler struct {
	k8sclient.Client
	Log     *zap.SugaredLogger
	Scheme  *k8sruntime.Scheme
	Scraper string
}

const controllerName = "metricsbinding"

// SetupWithManager creates controller for the MetricsBinding
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	return k8scontroller.NewControllerManagedBy(mgr).For(&vzapi.MetricsBinding{}).Complete(r)
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application wls is now manually managed.
func (r *Reconciler) Reconcile(ctx context.Context, req k8scontroller.Request) (k8scontroller.Result, error) {

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Metrics binding resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	metricsBinding := vzapi.MetricsBinding{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, &metricsBinding); err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("metricsbinding", req.NamespacedName, &metricsBinding)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for metrics binding resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling metrics binding resource %v, generation %v", req.NamespacedName, metricsBinding.Generation)

	res, err := r.doReconcile(ctx, metricsBinding, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	if err != nil {
		return clusters.NewRequeueWithDelay(), err
	}
	log.Oncef("Finished reconciling metrics binding %v", req.NamespacedName)

	return k8scontroller.Result{}, nil
}

// doReconcile performs the reconciliation operations for the ingress trait
func (r *Reconciler) doReconcile(ctx context.Context, metricsBinding vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	// Reconcile based on the status of the deletion timestamp
	if metricsBinding.GetDeletionTimestamp().IsZero() {
		return r.reconcileBindingCreateOrUpdate(ctx, &metricsBinding, log)
	}
	return r.reconcileBindingDelete(ctx, &metricsBinding, log)
}
