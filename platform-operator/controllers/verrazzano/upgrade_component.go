// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ComponentUpgradeState identifies the state of a component during upgrade
type ComponentUpgradeState string

const (
	// start is the state when a component is starting the upgrade flow
	start ComponentUpgradeState = "start"

	// doPreUpgrade is the state when a component does pre-upgrade
	doPreUpgrade ComponentUpgradeState = "doPreUpgrade"

	// doUpgrade is the state where a component does upgrade
	doUpgrade ComponentUpgradeState = "doUpgrade"

	// waitReady is the state when a component is waiting for upgrade ready
	waitReady ComponentUpgradeState = "waitReady"

	// doPostUpgrade is the state when a component is doing post upgrade
	doPostUpgrade ComponentUpgradeState = "doPostUpgrade"

	// upgradeDone is the state when component upgrade is done
	upgradeDone ComponentUpgradeState = "upgradeDone"
)

// componentUpgradeContext has the upgrade context for a Verrazzano component upgrade
type componentUpgradeContext struct {
	state ComponentUpgradeState
}

// upgradeComponents will upgrade the components as required
func (r *Reconciler) upgradeComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, tracker *upgradeTracker) (ctrl.Result, error) {
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
		upgradeContext := tracker.getComponentUpgradeContext(compName)

		switch upgradeContext.state {
		case start:
			// Check if component is installed, if not continue
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			if !installed {
				compLog.Oncef("Component %s is not installed; upgrade being skipped", compName)
				continue
			}
			upgradeContext.state = doPreUpgrade

		case doPreUpgrade:
			compLog.Oncef("Component %s pre-upgrade running", compName)
			if err := comp.PreUpgrade(compContext); err != nil {
				// for now, this will be fatal until upgrade is retry-able
				return ctrl.Result{}, err
			}
			upgradeContext.state = doPreUpgrade

		case doUpgrade:
			compLog.Oncef("Component %s upgrade running", compName)
			if err := comp.Upgrade(compContext); err != nil {
				compLog.Errorf("Error upgrading component %s: %v", compName, err)
				// check to see whether this is due to a pending upgrade
				r.resolvePendingUpgrades(compName, compLog)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
			}
			upgradeContext.state = waitReady

		case waitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been upgraded. Waiting for the component to be ready", compName)
				return newRequeueWithDelay(), nil
			}
			upgradeContext.state = doPostUpgrade

		case doPostUpgrade:
			compLog.Oncef("Component %s post-upgrade running", compName)
			if err := comp.PostUpgrade(compContext); err != nil {
				// for now, this will be fatal until upgrade is retry-able
				return ctrl.Result{}, err
			}
			upgradeContext.state = upgradeDone

		case upgradeDone:
			compLog.Oncef("Component %s has successfully upgraded and is ready", compName)
			continue
		}
	}
	return ctrl.Result{}, nil
}

// getComponentUpgradeContext gets the upgrade context for the component
func (vuc *upgradeTracker) getComponentUpgradeContext(compName string) *componentUpgradeContext {
	cuc, ok := vuc.compMap[compName]
	if !ok {
		cuc = &componentUpgradeContext{
			state: start,
		}
		vuc.compMap[compName] = cuc
	}
	return cuc
}
