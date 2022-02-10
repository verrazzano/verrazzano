// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"strconv"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile upgrade will upgrade the components as required
func (r *Reconciler) reconcileUpgrade(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Oncef("Upgrading Verrazzano to version %s", cr.Spec.Version)

	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

	// Only write the upgrade started message once
	if !isLastCondition(cr.Status, installv1alpha1.CondUpgradeStarted) {
		err := r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
			installv1alpha1.CondUpgradeStarted)
		// Always requeue to get a fresh copy of status and avoid potential conflict
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	spiCtx, err := spi.NewContext(log, r, cr, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Loop through all of the Verrazzano components and upgrade each one sequentially
	// - for now, upgrade is blocking
	for _, comp := range registry.GetComponents() {
		compName := comp.Name()
		compContext := spiCtx.Init(compName).Operation(vzconst.UpgradeOperation)
		compLog := compContext.Log()

		installed, err := comp.IsInstalled(compContext)
		if err != nil {
			return newRequeueWithDelay(), err
		}
		if !installed {
			compLog.Oncef("Component %s is not installed; upgrade being skipped", compName)
			continue
		}
		compLog.Oncef("Component %s pre-upgrade running", compName)
		if err := comp.PreUpgrade(compContext); err != nil {
			// for now, this will be fatal until upgrade is retry-able
			return ctrl.Result{}, err
		}
		compLog.Oncef("Component %s upgrade running", compName)
		if err := comp.Upgrade(compContext); err != nil {
			compLog.Errorf("Error upgrading component %s: %v", compName, err)
			msg := fmt.Sprintf("Error upgrading component %s - %s\".  Error is %s", compName,
				fmtGeneration(cr.Generation), err.Error())
			err := r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeFailed)
			return ctrl.Result{}, err
		}
		compLog.Oncef("Component %s post-upgrade running", compName)
		if err := comp.PostUpgrade(compContext); err != nil {
			// for now, this will be fatal until upgrade is retry-able
			return ctrl.Result{}, err
		}
	}

	// Invoke the global post upgrade function after all components are upgraded.
	err = postUpgrade(log, r)
	if err != nil {
		log.Errorf("Error running Verrazzano system-level post-upgrade")
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	msg := fmt.Sprintf("Verrazzano successfully upgraded to version %s", cr.Spec.Version)
	log.Info(msg)
	cr.Status.Version = targetVersion
	if err = r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeComplete); err != nil {
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{}, nil
}

// Return true if Verrazzano is installed
func isInstalled(st installv1alpha1.VerrazzanoStatus) bool {
	for _, cond := range st.Conditions {
		if cond.Type == installv1alpha1.CondInstallComplete {
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

func fmtGeneration(gen int64) string {
	s := strconv.FormatInt(gen, 10)
	return "generation:" + s
}

func postUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client) error {
	return istio.RestartComponents(log, config.GetInjectedSystemNamespaces(), client)
}
