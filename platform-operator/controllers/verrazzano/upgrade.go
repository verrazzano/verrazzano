// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"

	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// vzStart is the state where Verrazzano is starting the upgrade flow
	vzStart VerrazzanoUpgradeState = "vzStart"

	// vzUpgradeComponents is the state where the components are being upgraded
	vzUpgradeComponents VerrazzanoUpgradeState = "vzUpgradeComponents"

	// vzDoPostUpgrade is the state where Verrazzano is doing a post-upgrade
	vzDoPostUpgrade VerrazzanoUpgradeState = "vzDoPostUpgrade"

	// vzWaitPostUpgradeDone is the state when Verrazzano is waiting for postUpgrade to be done
	vzWaitPostUpgradeDone VerrazzanoUpgradeState = "vzWaitPostUpgradeDone"

	// vzUpgradeDone is the state when upgrade is done
	vzUpgradeDone VerrazzanoUpgradeState = "vzUpgradeDone"
)

// ComponentUpgradeState identifies the state of a component during upgrade
type VerrazzanoUpgradeState string

// upgradeTracker has the upgrade context for the Verrazzano upgrade
type upgradeTracker struct {
	vzState VerrazzanoUpgradeState
	gen     int64
	compMap map[string]*componentUpgradeContext
}

// upgradeTrackerMap has a map of upgradeTrackers, one entry per Verrazzano CR resource generation
var upgradeTrackerMap = make(map[string]*upgradeTracker)

// reconcileUpgrade will reconcile
func (r *Reconciler) reconcileUpgrade(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Oncef("Upgrading Verrazzano to version %s", cr.Spec.Version)

	tracker := getUpgradeTracker(cr)

	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

	switch tracker.vzState {
	case vzStart:
		// Only write the upgrade started message once
		if !isLastCondition(cr.Status, installv1alpha1.CondUpgradeStarted) {
			err := r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
				installv1alpha1.CondUpgradeStarted)
			// Always requeue to get a fresh copy of status and avoid potential conflict
			return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
		}
		tracker.vzState = vzUpgradeComponents

	case vzUpgradeComponents:
		// Upgrade the components
		res, err := r.upgradeComponents(log, cr, getUpgradeTracker(cr))
		if err != nil || res.Requeue {
			return res, err
		}
		tracker.vzState = vzDoPostUpgrade

	case vzDoPostUpgrade:
		// Invoke the global post upgrade function after all components are upgraded.
		err := postVerrazzanoUpgrade(log, r)
		if err != nil {
			log.Errorf("Error running Verrazzano system-level post-upgrade")
			return newRequeueWithDelay(), err
		}
		tracker.vzState = vzWaitPostUpgradeDone

	case vzWaitPostUpgradeDone:
		spiCtx, err := spi.NewContext(log, r, cr, r.DryRun)
		if err != nil {
			return newRequeueWithDelay(), err
		}
		for _, comp := range registry.GetComponents() {
			compName := comp.Name()
			compContext := spiCtx.Init(compName).Operation(vzconst.UpgradeOperation)
			if !comp.IsReady(compContext) {
				log.Progressf("Post-upgrade is waiting for all components to be ready.  Component %s is not yet ready", compName)
				return newRequeueWithDelay(), nil
			}
		}
		tracker.vzState = vzUpgradeDone

	case vzUpgradeDone:
		msg := fmt.Sprintf("Verrazzano successfully upgraded to version %s", cr.Spec.Version)
		log.Once(msg)
		cr.Status.Version = targetVersion
		if err := r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeComplete); err != nil {
			return newRequeueWithDelay(), err
		}
		// Upgrade completely done
		deleteUpgradeTracker(cr)
		return ctrl.Result{}, nil
	}

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
	log.Oncef("Checking if any pods with Istio sidecars need to be restarted")
	return istio.RestartComponents(log, config.GetInjectedSystemNamespaces(), client)
}

// getNSNKey gets the key for the verrazzano resource
func getNSNKey(cr *installv1alpha1.Verrazzano) string {
	return fmt.Sprintf("%s-%s", cr.Namespace, cr.Name)
}

// getUpgradeTracker gets the upgrade tracker for Verrazzano
func getUpgradeTracker(cr *installv1alpha1.Verrazzano) *upgradeTracker {
	key := getNSNKey(cr)
	vuc, ok := upgradeTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &upgradeTracker{
			vzState: vzStart,
			gen:     cr.Generation,
			compMap: make(map[string]*componentUpgradeContext),
		}
		upgradeTrackerMap[key] = vuc
	}
	return vuc
}

// deleteUpgradeTracker deletes the upgrade tracker for the Verrazzano resource
func deleteUpgradeTracker(cr *installv1alpha1.Verrazzano) {
	key := getNSNKey(cr)
	_, ok := upgradeTrackerMap[key]
	if ok {
		delete(upgradeTrackerMap, key)
	}
}
