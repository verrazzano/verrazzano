// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/selection"

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
	// vzStateStart is the state where Verrazzano is starting the upgrade flow
	vzStateStart VerrazzanoUpgradeState = "vzStart"

	// vzStateUpgradeComponents is the state where the components are being upgraded
	vzStateUpgradeComponents VerrazzanoUpgradeState = "vzUpgradeComponents"

	// vzStatePostUpgrade is the state where Verrazzano is doing a post-upgrade
	vzStatePostUpgrade VerrazzanoUpgradeState = "vzDoPostUpgrade"

	// vzStateWaitPostUpgradeDone is the state when Verrazzano is waiting for postUpgrade to be done
	vzStateWaitPostUpgradeDone VerrazzanoUpgradeState = "vzWaitPostUpgradeDone"

	// vzStateUpgradeDone is the state when upgrade is done
	vzStateUpgradeDone VerrazzanoUpgradeState = "vzUpgradeDone"

	// vzStateEnd is the terminal state
	vzStateEnd VerrazzanoUpgradeState = "vzStateEnd"
)

// VerrazzanoUpgradeState identifies the state of a Verrazzano upgrade operation
type VerrazzanoUpgradeState string

// upgradeTracker has the upgrade context for the Verrazzano upgrade
// This tracker keeps an in-memory upgrade state for Verrazzano and the components that
// are being upgrade.
type upgradeTracker struct {
	vzState VerrazzanoUpgradeState
	gen     int64
	compMap map[string]*componentUpgradeContext
}

// upgradeTrackerMap has a map of upgradeTrackers, one entry per Verrazzano CR resource generation
var upgradeTrackerMap = make(map[string]*upgradeTracker)

// reconcileUpgrade will upgrade a Verrazzano installation
func (r *Reconciler) reconcileUpgrade(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Oncef("Upgrading Verrazzano to version %s", cr.Spec.Version)

	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

	tracker := getUpgradeTracker(cr)
	for tracker.vzState != vzStateEnd {
		switch tracker.vzState {
		case vzStateStart:
			// Only write the upgrade started message once
			if !isLastCondition(cr.Status, installv1alpha1.CondUpgradeStarted) {
				err := r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
					installv1alpha1.CondUpgradeStarted)
				// Always requeue to get a fresh copy of status and avoid potential conflict
				return newRequeueWithDelay(), err
			}
			tracker.vzState = vzStateUpgradeComponents

		case vzStateUpgradeComponents:
			// Upgrade the components
			log.Once("Upgrading all Verrazzano components")
			res, err := r.upgradeComponents(log, cr, tracker)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = vzStatePostUpgrade

		case vzStatePostUpgrade:
			// Invoke the global post upgrade function after all components are upgraded.
			log.Once("Doing Verrazzano post-upgrade processing")
			err := postVerrazzanoUpgrade(log, r, cr)
			if err != nil {
				log.Errorf("Error running Verrazzano system-level post-upgrade")
				return newRequeueWithDelay(), err
			}
			tracker.vzState = vzStateWaitPostUpgradeDone

		case vzStateWaitPostUpgradeDone:
			log.Progress("Post-upgrade is waiting for all components to be ready")
			spiCtx, err := spi.NewContext(log, r, cr, r.DryRun)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			// Check installed enabled component and make sure it is ready
			for _, comp := range registry.GetComponents() {
				compName := comp.Name()
				compContext := spiCtx.Init(compName).Operation(vzconst.UpgradeOperation)
				installed, err := comp.IsInstalled(compContext)
				if err != nil {
					return ctrl.Result{}, err
				}
				if installed && !comp.IsReady(compContext) {
					log.Progressf("Waiting for component %s to be ready after post-upgrade", compName)
					return newRequeueWithDelay(), nil
				}
				log.Oncef("Component %s is ready after post-upgrade", compName)

			}
			tracker.vzState = vzStateUpgradeDone

		case vzStateUpgradeDone:
			msg := fmt.Sprintf("Verrazzano successfully upgraded to version %s", cr.Spec.Version)
			log.Once(msg)
			cr.Status.Version = targetVersion
			for _, comp := range registry.GetComponents() {
				compName := comp.Name()
				componentStatus := cr.Status.Components[compName]
				if componentStatus != nil {
					log.Oncef("Component %s has been upgraded from generation %v to %v %v", compName, componentStatus.LastReconciledGeneration, cr.Generation, componentStatus.State)
					componentStatus.LastReconciledGeneration = cr.Generation
				}
			}
			if err := r.updateStatus(log, cr, msg, installv1alpha1.CondUpgradeComplete); err != nil {
				return newRequeueWithDelay(), err
			}
			// Upgrade completely done
			deleteUpgradeTracker(cr)
			tracker.vzState = vzStateEnd
		}
	}
	// Upgrade done, no need to requeue
	return ctrl.Result{}, nil
}

// resolvePendingUpgrdes will delete any helm secrets with a status other than "deployed" for the given component
func (r *Reconciler) resolvePendingUpgrades(compName string, compLog vzlog.VerrazzanoLogger) {
	nameReq, _ := kblabels.NewRequirement("name", selection.Equals, []string{compName})
	statusReq, _ := kblabels.NewRequirement("status", selection.NotEquals, []string{"deployed"})
	labelSelector := kblabels.NewSelector()
	labelSelector = labelSelector.Add(*nameReq, *statusReq)
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
		compLog.Infof("%s labels:", helmSecrets.Items[i].Name)
		for k, v := range helmSecrets.Items[i].Labels {
			compLog.Infof("key: %s, value: %s", k, v)
		}
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
func postVerrazzanoUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client, cr *installv1alpha1.Verrazzano) error {
	log.Oncef("Checking if any pods with Istio sidecars need to be restarted to pick up the new version of the Istio proxy")
	return istio.RestartComponents(log, config.GetInjectedSystemNamespaces(), cr.Generation)
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
			vzState: vzStateStart,
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
