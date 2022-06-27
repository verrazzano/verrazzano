// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// vzStateUninstallStart is the state where Verrazzano is starting the uninstall flow
	vzStateUninstallStart uninstallState = "vzStateUninstallStart"

	// vzStateUninstallRancherLocal is the state where the Rancher local cluster is being uninstalled
	vzStateUninstallRancherLocal uninstallState = "vzStateUninstallRancherLocal"

	// vzStateUninstallMC is the state where the multi-cluster resources are being uninstalled
	vzStateUninstallMC uninstallState = "vzStateUninstallMC"

	// vzStateUninstallComponents is the state where the components are being uninstalled
	vzStateUninstallComponents uninstallState = "vzStateUninstallComponents"

	// vzStateUninstallDone is the state when uninstall is done
	vzStateUninstallDone uninstallState = "vzStateUninstallDone"

	// vzStateUninstallEnd is the terminal state
	vzStateUninstallEnd uninstallState = "vzStateUninstallEnd"
)

// uninstallState identifies the state of a Verrazzano uninstall operation
type uninstallState string

// UninstallTracker has the Uninstall context for the Verrazzano Uninstall
// This tracker keeps an in-memory Uninstall state for Verrazzano and the components that
// are being Uninstall.
type UninstallTracker struct {
	vzState uninstallState
	gen     int64
	compMap map[string]*componentUninstallContext
}

// UninstallTrackerMap has a map of UninstallTrackers, one entry per Verrazzano CR resource generation
var UninstallTrackerMap = make(map[string]*UninstallTracker)

// reconcileUninstall will Uninstall a Verrazzano installation
func (r *Reconciler) reconcileUninstall(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Oncef("Uninstalling Verrazzano %s/%s", cr.Namespace, cr.Name)

	tracker := getUninstallTracker(cr)
	done := false
	for !done {
		switch tracker.vzState {
		case vzStateUninstallStart:
			tracker.vzState = vzStateUninstallRancherLocal

		case vzStateUninstallRancherLocal:
			tracker.vzState = vzStateUninstallMC

		case vzStateUninstallMC:
			tracker.vzState = vzStateUninstallComponents

		case vzStateUninstallComponents:
			log.Once("Uninstalling all Verrazzano components")
			res, err := r.uninstallComponents(log, cr, tracker)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = vzStateUninstallDone

		case vzStateUninstallDone:
			tracker.vzState = vzStateUninstallEnd

		case vzStateUninstallEnd:
			done = true
			// Uninstall completely done
			deleteUninstallTracker(cr)
		}
	}
	// Uninstall done, no need to requeue
	return ctrl.Result{}, nil
}

// getUninstallTracker gets the Uninstall tracker for Verrazzano
func getUninstallTracker(cr *installv1alpha1.Verrazzano) *UninstallTracker {
	key := getNSNKey(cr)
	vuc, ok := UninstallTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &UninstallTracker{
			vzState: vzStateUninstallStart,
			gen:     cr.Generation,
			compMap: make(map[string]*componentUninstallContext),
		}
		UninstallTrackerMap[key] = vuc
	}
	return vuc
}

// deleteUninstallTracker deletes the Uninstall tracker for the Verrazzano resource
func deleteUninstallTracker(cr *installv1alpha1.Verrazzano) {
	key := getNSNKey(cr)
	_, ok := UninstallTrackerMap[key]
	if ok {
		delete(UninstallTrackerMap, key)
	}
}
