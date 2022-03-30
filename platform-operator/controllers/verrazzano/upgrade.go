// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/controller"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	kblabels "k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentUpgradeState identifies the state of a component during upgrade
type ComponentUpgradeState string

const (
	// StateInit is the state when a component is starting the upgrade flow
	StateInit ComponentUpgradeState = "Init"

	// StatePreUpgrading is the state when a component has successfully called component preUpgrade
	StatePreUpgrading ComponentUpgradeState = "PreUpgrading"

	// StateUpgrading is the state when a component has successfully called component Upgrade
	StateUpgrading ComponentUpgradeState = "Upgrading"
)

// upgradeTracker has the upgrade context for the Verrazzano upgrade
type upgradeTracker struct {
	gen     int64
	compMap map[string]*componentUpgradeContext
}

// componentUpgradeContext has the upgrade context for a Verrazzano component upgrade
type componentUpgradeContext struct {
	state ComponentUpgradeState
}

// upgradeTrackerMap has a map of upgradeTrackers, one entry per Verrazzano CR resource generation
var upgradeTrackerMap = make(map[string]*upgradeTracker)

// reconcileUpgrade will upgrade the components as required
func (r *Reconciler) reconcileUpgrade(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Get the upgradeTracker for this Verrazzano CR generation
	upgradeTracker := getUpgradeTracker(cr)

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
		upgradeContext := upgradeTracker.getComponentUpgradeContext(compName)

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
				// check to see whether this is due to a pending upgrade
				r.resolvePendingUpgrades(compName, compLog)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
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
	log.Oncef("Checking if any pods with Istio sidecars need to be restarted")
	err = postVerrazzanoUpgrade(log, r)
	if err != nil {
		log.Errorf("Error running Verrazzano system-level post-upgrade")
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	msg := fmt.Sprintf("Verrazzano successfully upgraded to version %s", cr.Spec.Version)
	log.Once(msg)
	cr.Status.Version = targetVersion
	if err = r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeComplete); err != nil {
		return newRequeueWithDelay(), err
	}
	// Upgrade completely done
	deleteUpgradeTracker(cr)
	return ctrl.Result{}, nil
}

// resolvePendingUpgrdes will delete any helm secrets with a "pending-upgrade" status for the given component
func (r *Reconciler) resolvePendingUpgrades(compName string, compLog vzlog.VerrazzanoLogger) {
	labelSelector := kblabels.Set{"name": compName, "status": "pending-upgrade"}.AsSelector()
	helmSecrets := v1.SecretList{}
	err := r.Client.List(context.TODO(), &helmSecrets, &clipkg.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		if errors.IsNotFound(err) {
			compLog.Debugf("No pending upgrade found for component %s.  Re-trying upgrade", compName)
		} else {
			compLog.Errorf("Error attempting to determine if upgrade is pending for component %s: %v.  Re-trying upgrade", compName, err)
		}
	}
	// remove any pending upgrade secrets
	for i := range helmSecrets.Items {
		err := r.Client.Delete(context.TODO(), &helmSecrets.Items[i], &clipkg.DeleteOptions{})
		if err != nil {
			compLog.Errorf("Unable to remove pending upgrade helm secret for component %s: %v", compName, err)
		} else {
			compLog.Infof("Resolved pending upgrade for component %s", compName)
		}
	}
}

// isInstalled returns true if Verrazzano is installed
func isInstalled(st installv1alpha1.VerrazzanoStatus) bool {
	for _, cond := range st.Conditions {
		if cond.Type == installv1alpha1.CondInstallComplete {
			return true
		}
	}
	return false
}

// isLastCondition returns true if the last condition matches the condition type
func isLastCondition(st installv1alpha1.VerrazzanoStatus, conditionType installv1alpha1.ConditionType) bool {
	l := len(st.Conditions)
	if l == 0 {
		return false
	}
	return st.Conditions[l-1].Type == conditionType
}

// postVerrazzanoUpgrade restarts pods with old Istio sidecar proxies
func postVerrazzanoUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client) error {
	return istio.RestartComponents(log, config.GetInjectedSystemNamespaces(), client)
}

// getNsnKey gets the key for the verrazzano resource
func getNsnKey(cr *installv1alpha1.Verrazzano) string {
	return fmt.Sprintf("%s-%s", cr.Namespace, cr.Name)
}

// getUpgradeTracker gets the upgrade tracker for Verrazzano
func getUpgradeTracker(cr *installv1alpha1.Verrazzano) *upgradeTracker {
	key := getNsnKey(cr)
	vuc, ok := upgradeTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &upgradeTracker{
			gen:     cr.Generation,
			compMap: make(map[string]*componentUpgradeContext),
		}
		upgradeTrackerMap[key] = vuc
	}
	return vuc
}

// deleteUpgradeTracker deletes the upgrade tracker for the Verrazzano resource
func deleteUpgradeTracker(cr *installv1alpha1.Verrazzano) {
	key := getNsnKey(cr)
	_, ok := upgradeTrackerMap[key]
	if ok {
		delete(upgradeTrackerMap, key)
	}
}

// getComponentUpgradeContext gets the upgrade context for the component
func (vuc *upgradeTracker) getComponentUpgradeContext(compName string) *componentUpgradeContext {
	cuc, ok := vuc.compMap[compName]
	if !ok {
		cuc = &componentUpgradeContext{
			state: StateInit,
		}
		vuc.compMap[compName] = cuc
	}
	return cuc
}
