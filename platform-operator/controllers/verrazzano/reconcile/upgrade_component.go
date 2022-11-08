// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

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

// componentUpgradeState identifies the state of a component during upgrade
type componentUpgradeState string

const (
	// compStateUpgradeInit is the state when a component is starting the upgrade flow
	compStateUpgradeInit componentUpgradeState = "componentStateUpgradeInit"

	// compStatePreUpgrade is the state when a component does a pre-upgrade
	compStatePreUpgrade componentUpgradeState = "compStatePreUpgrade"

	// compStateUpgrade is the state where a component does an upgrade
	compStateUpgrade componentUpgradeState = "compStateUpgrade"

	// compStateUpgradeWaitReady is the state when a component is waiting for upgrade ready
	compStateUpgradeWaitReady componentUpgradeState = "compStateUpgradeWaitReady"

	// compStatePostUpgrade is the state when a component is doing a post-upgrade
	compStatePostUpgrade componentUpgradeState = "compStatePostUpgrade"

	// compStateUpgradeDone is the state when component upgrade is done
	compStateUpgradeDone componentUpgradeState = "compStateUpgradeDone"

	// compStateUpgradeEnd is the terminal state
	compStateUpgradeEnd componentUpgradeState = "compStateEnd"
)

// componentUpgradeContext has the upgrade context for a Verrazzano component upgrade
type componentUpgradeContext struct {
	state componentUpgradeState
}

// upgradeComponents will upgrade the components as required
func (r *Reconciler) upgradeComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, tracker *upgradeTracker) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Loop through all of the Verrazzano components and upgrade each one.
	// Don't move to the next component until the current one has been succcessfully upgraded
	for _, comp := range registry.GetComponents() {
		upgradeContext := tracker.getComponentUpgradeContext(comp.Name())
		result, err := r.upgradeSingleComponent(spiCtx, upgradeContext, comp)
		if err != nil || result.Requeue {
			return result, err
		}

	}
	// All components have been upgraded
	return ctrl.Result{}, nil
}

// upgradeSingleComponent upgrades a single component
func (r *Reconciler) upgradeSingleComponent(spiCtx spi.ComponentContext, upgradeContext *componentUpgradeContext, comp spi.Component) (ctrl.Result, error) {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.UpgradeOperation)
	compLog := compContext.Log()

	for upgradeContext.state != compStateUpgradeEnd {
		switch upgradeContext.state {
		case compStateUpgradeInit:
			// Check if component is installed, if not continue
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s is installed: %v", compName, err)
				return ctrl.Result{}, err
			}
			if installed {
				compLog.Oncef("Component %s is installed and will be upgraded", compName)
				if err := r.updateComponentStatus(compContext, "Upgrade started", installv1alpha1.CondUpgradeStarted); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				upgradeContext.state = compStatePreUpgrade
			} else {
				compLog.Oncef("Component %s is not installed; upgrade being skipped", compName)
				upgradeContext.state = compStateUpgradeEnd
			}

		case compStatePreUpgrade:
			compLog.Oncef("Component %s pre-upgrade running", compName)
			if err := comp.PreUpgrade(compContext); err != nil {
				return ctrl.Result{}, err
			}
			upgradeContext.state = compStateUpgrade

		case compStateUpgrade:
			compLog.Progressf("Component %s upgrade running", compName)
			if err := comp.Upgrade(compContext); err != nil {
				compLog.Errorf("Failed upgrading component %s, will retry: %v", compName, err)
				// check to see whether this is due to a pending upgrade
				r.resolvePendingUpgrades(compName, compLog)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
			}
			upgradeContext.state = compStateUpgradeWaitReady

		case compStateUpgradeWaitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been upgraded. Waiting for the component to be ready", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Progressf("Component %s is ready after being upgraded", compName)
			upgradeContext.state = compStatePostUpgrade

		case compStatePostUpgrade:
			compLog.Oncef("Component %s post-upgrade running", compName)
			if err := comp.PostUpgrade(compContext); err != nil {
				return ctrl.Result{}, err
			}
			upgradeContext.state = compStateUpgradeDone

		case compStateUpgradeDone:
			compLog.Oncef("Component %s has successfully upgraded", compName)
			if err := r.updateComponentStatus(compContext, "Upgrade complete", installv1alpha1.CondUpgradeComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			upgradeContext.state = compStateUpgradeEnd
		}
	}
	// Component has been upgraded
	return ctrl.Result{}, nil
}

// getComponentUpgradeContext gets the upgrade context for the component
func (vuc *upgradeTracker) getComponentUpgradeContext(compName string) *componentUpgradeContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentUpgradeContext{
			state: compStateUpgradeInit,
		}
		vuc.compMap[compName] = context
	}
	return context
}
