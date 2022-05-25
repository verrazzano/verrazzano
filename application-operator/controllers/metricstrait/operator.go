// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// doOperatorReconcile reconciles a metrics trait to work with the Prometheus Operator
// This reconciler will create a ServiceMonitor for each metrics trait application to hook up metrics with Prometheus
func (r *Reconciler) doOperatorReconcile(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	if trait.DeletionTimestamp.IsZero() {
		return r.reconcileOperatorTraitCreateOrUpdate(ctx, trait, log)
	}
	return r.reconcileOperatorTraitDelete(ctx, trait, log)
}

func (r *Reconciler) reconcileOperatorTraitCreateOrUpdate(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	var err error
	// Add finalizer if required.
	if err := r.addFinalizerIfRequired(ctx, trait, log); err != nil {
		return reconcile.Result{}, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	if workload, err = vznav.FetchWorkloadFromTrait(ctx, r, log, trait); err != nil || workload == nil {
		return reconcile.Result{}, err
	}

	// Construct the trait defaults from the trait and the workload resources
	traitDefaults, supported, err := r.fetchTraitDefaults(ctx, workload, log)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !supported || traitDefaults == nil {
		return reconcile.Result{Requeue: false}, nil
	}

	// Find the child resources of the workload based on the childResourceKinds from the
	// workload definition, workload uid and the ownerReferences of the children.
	var children []*unstructured.Unstructured
	if children, err = vznav.FetchWorkloadChildren(ctx, r, log, workload); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the related resources of the trait and collect the outcomes.
	status := r.createOrUpdateRelatedWorkloads(ctx, trait, workload, traitDefaults, children, log)

	status.RecordOutcome(r.updateServiceMonitor(ctx, trait, workload, traitDefaults, log))

	return r.updateTraitStatus(ctx, trait, status, log)
}

func (r *Reconciler) reconcileOperatorTraitDelete(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	status := r.deleteOrUpdateObsoleteResources(ctx, trait, &reconcileresults.ReconcileResults{}, log)
	// Only remove the finalizer if all related resources were successfully updated.
	if !status.ContainsErrors() {
		if err := r.removeFinalizerIfRequired(ctx, trait, log); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// fetchTraitDefaults fetches metrics trait default values.
// These default values are workload type dependent.
func (r *Reconciler) fetchTraitDefaults(ctx context.Context, workload *unstructured.Unstructured, log vzlog.VerrazzanoLogger) (*vzapi.MetricsTraitSpec, bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return nil, true, log.ErrorfNewErr("Failed to get the API version from the workload: %v", err)
	}

	workloadType := GetSupportedWorkloadType(apiVerKind)
	switch workloadType {
	case constants.WorkloadTypeWeblogic:
		spec, err := r.NewTraitDefaultsForWLSDomainWorkload(ctx, workload)
		return spec, true, err
	case constants.WorkloadTypeCoherence:
		spec, err := r.NewTraitDefaultsForCOHWorkload(ctx, workload)
		return spec, true, err
	case constants.WorkloadTypeGeneric:
		spec, err := r.NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	default:
		// Log the kind/workload is unsupported and return a nil trait.
		log.Debugf("unsupported kind %s of workload %s", apiVerKind, vznav.GetNamespacedNameFromUnstructured(workload))
		return nil, false, nil
	}

}
