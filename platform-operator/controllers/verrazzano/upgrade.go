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

// ComponentUpgradeState identifies the state of a component during upgrade
type ComponentUpgradeState string

const (
	// StateInit is the state when a Verrazzano component is starting the upgrade flow
	StateInit ComponentUpgradeState = "Init"

	// StatePreUpgrading is the state when a Verrazzano component has successfully called component preUpgrade
	StatePreUpgrading ComponentUpgradeState = "PreUpgrading"

	// StateUpgradeing is the state when a Verrazzano component has successfully called component Upgrade
	StateUpgrading ComponentUpgradeState = "Upgrading"

	// PostUpgradeCalled is the state when a Verrazzano component has successfully called component postUpgrade
	StatePostUpgrading ComponentUpgradeState = "PostUpgrading"

	// StateUpgraded is the state when a Verrazzano component has successfully upgraded the component
	StateUpgraded ComponentUpgradeState = "Upgraded"
)

// upgradeContext has a map of upgradeContexts, one entry per verrazzano CR resource generation
var upgradeContextMap = make(map[string]*vzUpgradeContext)

// vzUpgradeContext has the upgrade context for the Verrazzano upgrade
type vzUpgradeContext struct {
	gen     int64
	compMap map[string]*componentUpgradeContext
}

// componentUpgradeContext has the upgrade context for the Verrazzano upgrade
type componentUpgradeContext struct {
	state ComponentUpgradeState
}

// Reconcile upgrade will upgrade the components as required
func (r *Reconciler) reconcileUpgrade(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Get the upgradeContext for this Verrazzano CR generation
	vuc := getUpgradeContext(cr)

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
		upgradeContext := vuc.getUpgradeContext(compName)

		if upgradeContext.state == StateInit {
			// Check if component is installed, if not continue
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
			upgradeContext.state = StatePreUpgrading
		}

		if upgradeContext.state == StatePreUpgrading {
			compLog.Oncef("Component %s upgrade running", compName)
			if err := comp.Upgrade(compContext); err != nil {
				compLog.Errorf("Error upgrading component %s: %v", compName, err)
				msg := fmt.Sprintf("Error upgrading component %s - %s\".  Error is %s", compName,
					fmtGeneration(cr.Generation), err.Error())
				err := r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeFailed)
				return ctrl.Result{}, err
			}
			upgradeContext.state = StateUpgrading
		}

		if !comp.IsReady(compContext) {
			compLog.Progressf("Component %s has been upgraded. Waiting for the component to be ready", compName)
			return newRequeueWithDelay(), nil
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
	// Upgrade completely done
	deleteUpgradeContext(cr)
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

// get the key for the verrazzano resource
func getNsnKey(cr *installv1alpha1.Verrazzano) string {
	return fmt.Sprintf("%s-%s", cr.Namespace, cr.Name)
}

// Get the upgrade context for Verrazzano
func getUpgradeContext(cr *installv1alpha1.Verrazzano) *vzUpgradeContext {
	key := getNsnKey(cr)
	vuc, ok := upgradeContextMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &vzUpgradeContext{
			gen:     cr.Generation,
			compMap: make(map[string]*componentUpgradeContext),
		}
		upgradeContextMap[key] = vuc
	}
	return vuc
}

// Delete the upgrade context for Verrazzano
func deleteUpgradeContext(cr *installv1alpha1.Verrazzano) {
	key := getNsnKey(cr)
	_, ok := upgradeContextMap[key]
	if ok {
		delete(upgradeContextMap, key)
	}
}

// Get the upgrade context for the component
func (vuc *vzUpgradeContext) getUpgradeContext(compName string) *componentUpgradeContext {
	cuc, ok := vuc.compMap[compName]
	if !ok {
		cuc = &componentUpgradeContext{
			state: StateInit,
		}
		vuc.compMap[compName] = cuc
	}
	return cuc
}
