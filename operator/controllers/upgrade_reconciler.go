// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"fmt"

	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/internal/component"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The max upgrade failures for a given upgrade attempt is 2
const failedUpgradeLimit = 2

// Reconcile upgrade will upgrade the components as required
func (r *VerrazzanoReconciler) reconcileUpgrade(log *zap.SugaredLogger, req ctrl.Request, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

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
		if r.DryRun {
			// Eventually, pass this down through Component.Upgrade() and into the helm command
			log.Info("Dry run enabled, skipping upgrade")
			break
		}
		err := comp.Upgrade(r, cr.Namespace)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error upgrading component %s", comp.Name()))
			msg := fmt.Sprintf("Error upgrading component %s.  Error is %s", comp.Name(), err.Error())
			err := r.updateStatus(log, cr, msg, installv1alpha1.UpgradeFailed)
			return ctrl.Result{}, err
		}
	}
	msg := fmt.Sprintf("Verrazzano upgraded to version %s successfully", cr.Spec.Version)
	cr.Status.Version = targetVersion
	err := r.updateStatus(log, cr, msg, installv1alpha1.UpgradeComplete)
	return ctrl.Result{}, err
}

// Return true if verrazzano is installed
func isInstalled(st installv1alpha1.VerrazzanoStatus) bool {
	for _, cond := range st.Conditions {
		if cond.Type == installv1alpha1.InstallComplete {
			return true
		}
	}
	return false
}

// Return true if the last condition matches the condition type
func isLastCondition(st installv1alpha1.VerrazzanoStatus, conditionType installv1alpha1.ConditionType) bool {
	l := len(st.Conditions)
	if l == 0 {
		return false
	}
	return st.Conditions[l-1].Type == conditionType
}

// Get the number of times an upgrade failed since last installation or successful upgrade
func upgradeFailureCount(st installv1alpha1.VerrazzanoStatus) int {
	var c int
	for _, cond := range st.Conditions {
		switch cond.Type {
		case installv1alpha1.UpgradeComplete:
			c = 0
		case installv1alpha1.UpgradeFailed:
			c++
		}
	}
	return c
}
