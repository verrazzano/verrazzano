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

// Reconcile upgrade will upgrade the components as required
func (r *VerrazzanoReconciler) reconcileUpgrade(log logr.Logger, req ctrl.Request, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	targetVersion := cr.Spec.Version
	err := validateVersion(targetVersion)
	if err != nil {
		log.Error(err, "Invalid upgrade version")
		return ctrl.Result{}, nil
	}

	r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
		installv1alpha1.UpgradeStarted)

	// Loop through all of the Verrazzano components and upgrade each one sequentially
	for _, comp := range component.GetComponents() {
		err := comp.Upgrade(cr.Namespace)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to upgrade component %s", comp.Name()))
			msg := fmt.Sprintf("Error upgrading component %s.  Error is %s", comp.Name(), err.Error())
			r.updateStatus(log, cr, msg, installv1alpha1.UpgradeFailed)

			// Do not requeue
			return ctrl.Result{}, nil
		}
	}
	msg := fmt.Sprintf("Verrazzano upgraded to version %s successfully", cr.Spec.Version)
	cr.Status.Version = targetVersion
	r.updateStatus(log, cr, msg, installv1alpha1.UpgradeComplete)
	return ctrl.Result{}, nil
}

// Validate the target version
func validateVersion(version string) error {
	// todo - do this in webhook and check that version matches chart version
	return nil
}
