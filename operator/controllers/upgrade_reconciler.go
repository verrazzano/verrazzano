// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"fmt"
	"github.com/go-logr/logr"
	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/internal/component"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The max upgrade failures for a given upgrade attempt is 2
const failedUpgradeLimit = 2

// Reconcile upgrade will upgrade the components as required
func (r *VerrazzanoReconciler) reconcileUpgrade(log logr.Logger, req ctrl.Request, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	targetVersion := cr.Spec.Version
	err := validateVersion(targetVersion)
	if err != nil {
		log.Error(err, "Invalid upgrade version")
		return ctrl.Result{}, nil
	}

	// Only allow upgrade to retry a certain amount of times during any upgrade attempt.
	if upgradeFailureCount(cr.Status) > failedUpgradeLimit {
		log.Info("Upgrade failure limit reached, upgrade will not be attempted")
		return ctrl.Result{}, nil
	}

	// Only write the upgrade started message once
	if !isLastCondition(cr.Status, installv1alpha1.UpgradeStarted) {
		r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
			installv1alpha1.UpgradeStarted)
		return ctrl.Result{Requeue: true}, nil
	}

	// Loop through all of the Verrazzano components and upgrade each one sequentially
	for _, comp := range component.GetComponents() {
		err := comp.Upgrade(cr.Namespace)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error upgrading component %s", comp.Name()))
			msg := fmt.Sprintf("Error upgrading component %s.  Error is %s", comp.Name(), err.Error())
			err := r.updateStatus(log, cr, msg, installv1alpha1.UpgradeFailed)
			return ctrl.Result{}, err
		}
	}
	msg := fmt.Sprintf("Verrazzano upgraded to version %s successfully", cr.Spec.Version)
	cr.Status.Version = targetVersion
	err = r.updateStatus(log, cr, msg, installv1alpha1.UpgradeComplete)
	return ctrl.Result{}, err
}

// Validate the target version
func validateVersion(version string) error {
	// todo - do this in webhook and check that version matches chart version
	return nil
}
