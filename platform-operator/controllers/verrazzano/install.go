// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	ctrl "sigs.k8s.io/controller-runtime"
)

// installTracker has the Install context for the Verrazzano Install
// This tracker keeps an in-memory Install state for Verrazzano and the components that
// are being Install.
type installTracker struct {
	gen     int64
	compMap map[string]*componentInstallContext
}

// installTrackerMap has a map of InstallTrackers, one entry per Verrazzano CR resource generation
var installTrackerMap = make(map[string]*installTracker)

// getUpgradeTrackerKey gets the tracker key for the Verrazzano resource
func getInstallTrackerKey(cr *vzapi.Verrazzano) string {
	return fmt.Sprintf("%s-%s-%s", cr.Namespace, cr.Name, string(cr.UID))
}

// getInstallTracker gets the install tracker for Verrazzano
func getInstallTracker(cr *vzapi.Verrazzano) *installTracker {
	key := getInstallTrackerKey(cr)
	vuc, ok := installTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &installTracker{
			gen:     cr.Generation,
			compMap: make(map[string]*componentInstallContext),
		}
		installTrackerMap[key] = vuc
	}
	return vuc
}

// deleteInstallTracker deletes the install tracker for the Verrazzano resource
func deleteInstallTracker(cr *vzapi.Verrazzano) {
	key := getInstallTrackerKey(cr)
	_, ok := installTrackerMap[key]
	if ok {
		delete(installTrackerMap, key)
	}
}

// reconcileComponents reconciles each component using the following rules:
// 1. Always requeue until all enabled components have completed installation
// 2. Don't update the component state until all the work in that state is done, since
//    that update will cause a state transition
// 3. Loop through all components before returning, except for the case
//    where update status fails, in which case we exit the function and requeue
//    immediately.
func (r *Reconciler) reconcileComponents(vzctx vzcontext.VerrazzanoContext, preUpgrade bool) (ctrl.Result, error) {

	spiCtx, err := spi.NewContext(vzctx.Log, r.Client, vzctx.ActualCR, nil, r.DryRun)
	if err != nil {
		spiCtx.Log().Errorf("Failed to create component context: %v", err)
		return newRequeueWithDelay(), err
	}

	cr := spiCtx.ActualCR()
	spiCtx.Log().Progress("Reconciling components for Verrazzano installation")

	tracker := getInstallTracker(cr)

	res, err := r.installComponents(spiCtx.Log(), cr, tracker, preUpgrade)
	if err != nil || res.Requeue {
		return res, err
	}

	deleteInstallTracker(cr)

	return ctrl.Result{}, nil
}
