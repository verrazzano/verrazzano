// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the Verrazzano CR.  This includes new installations, updates, upgrades, and partial uninstalls.
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	// Do CR initialization
	if res := r.initVzResource(log, actualCR); res.ShouldRequeue() {
		return res
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not modified the version in the Verrazzano CR to match the BOM.
	if upgradePending, err := r.isUpgradeRequired(actualCR); upgradePending || err != nil {
		spictx.Log.Oncef("Upgrade required before reconciling modules")
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	// Note: updating the VZ state to reconciling is done by the module shim to
	// avoid a long delay before the user sees any CR action.
	// See vzcomponent_status.go, UpdateVerrazzanoComponentStatus
	if r.isUpgrading(actualCR) {
		if err := r.updateStatusUpgrading(log, actualCR); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Do global pre-work
	if res := r.preWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do the actual install, update, and or upgrade.
	if res := r.doWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		if res := r.updateStatusIfNeeded(log, actualCR); res.ShouldRequeue() {
			return res
		}
		return res
	}

	// Do global post-work
	if res := r.postWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// All done reconciling.  Add the completed condition to the status and set the state back to Ready.
	r.updateStatusInstallUpgradeComplete(actualCR)
	return result.NewResult()
}
