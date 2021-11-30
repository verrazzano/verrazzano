// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"strconv"
	"strings"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// The max upgrade failures for a given upgrade attempt is 5
const failedUpgradeLimit = 5

// Reconcile upgrade will upgrade the components as required
func (r *Reconciler) reconcileUpgrade(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Debugf("enter reconcileUpgrade")

	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

	// Only allow upgrade to retry a certain amount of times during any upgrade attempt.
	if upgradeFailureCount(cr.Status, cr.Generation) > failedUpgradeLimit {
		log.Warn("Upgrade failure limit reached, upgrade will not be attempted")
		return ctrl.Result{}, nil
	}

	// Only write the upgrade started message once
	if !isLastCondition(cr.Status, installv1alpha1.UpgradeStarted) {
		err := r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
			installv1alpha1.UpgradeStarted)
		// Always requeue to get a fresh copy of status and avoid potential conflict
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	// Loop through all of the Verrazzano components and upgrade each one sequentially
	// - for now, upgrade is blocking
	for _, comp := range registry.GetComponents() {
		compName := comp.Name()
		log.Infof("Upgrading %s", compName)
		upgradeContext, err := spi.NewContext(log, r, cr, r.DryRun)
		if err != nil {
			return newRequeueWithDelay(), err
		}
		upgradeContext = upgradeContext.For(compName).Operation("upgrade")
		installed, err := comp.IsInstalled(upgradeContext)
		if err != nil {
			return newRequeueWithDelay(), err
		}
		if !installed {
			log.Infof("Skip upgrade for %s, not installed", compName)
			continue
		}
		log.Infof("Running pre-upgrade for %s", compName)
		if err := comp.PreUpgrade(upgradeContext); err != nil {
			// for now, this will be fatal until upgrade is retry-able
			return ctrl.Result{}, err
		}
		log.Infof("Running upgrade for %s", compName)
		if err := comp.Upgrade(upgradeContext); err != nil {
			log.Errorf("Error upgrading component %s: %v", compName, err)
			msg := fmt.Sprintf("Error upgrading component %s - %s\".  Error is %s", compName,
				fmtGeneration(cr.Generation), err.Error())
			err := r.updateStatus(log, cr, msg, installv1alpha1.UpgradeFailed)
			return ctrl.Result{}, err
		}
		log.Infof("Running post-upgrade for %s", compName)
		if err := comp.PostUpgrade(upgradeContext); err != nil {
			// for now, this will be fatal until upgrade is retry-able
			return ctrl.Result{}, err
		}
	}

	// Invoke the global post upgrade function after all components are upgraded.
	err := postUpgrade(log, r)
	if err != nil {
		log.Errorf("Error running Verrazzano system-level post-upgrade")
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	msg := fmt.Sprintf("Verrazzano upgraded to version %s successfully", cr.Spec.Version)
	log.Info(msg)
	cr.Status.Version = targetVersion
	if err = r.updateStatus(log, cr, msg, installv1alpha1.UpgradeComplete); err != nil {
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{}, nil
}

// Return true if Verrazzano is installed
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

// Get the number of times an upgrade failed for the specified
// generation, meaning the last time the CR spec was modified by the user
// This is needed as a circuit-breaker for upgrade and other operations where
// limit the retries on a given generation of the CR.
func upgradeFailureCount(st installv1alpha1.VerrazzanoStatus, generation int64) int {
	var c int
	for _, cond := range st.Conditions {
		// Look for an upgrade failed condition where the message contains the CR generation that
		// is currently being processed, then increment the count. If the generation is not found then
		// the condition is from a previous user upgrade request and we can ignore it.
		if cond.Type == installv1alpha1.UpgradeFailed &&
			strings.Contains(cond.Message, fmtGeneration(generation)) {
			c++
		}
	}
	return c
}

func fmtGeneration(gen int64) string {
	s := strconv.FormatInt(gen, 10)
	return "generation:" + s
}

func postUpgrade(log *zap.SugaredLogger, client clipkg.Client) error {
	return nil
}
