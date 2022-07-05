// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"

	clustersapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
			// If Rancher is installed, then delete local cluster
			found, comp := registry.FindComponent(rancher.ComponentName)
			if !found {
				tracker.vzState = vzStateUninstallMC
				continue
			}
			spiCtx, err := spi.NewContext(log, r.Client, cr, r.DryRun)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			compContext := spiCtx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation)
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			if !installed {
				tracker.vzState = vzStateUninstallMC
				continue
			}
			if err := rancher.DeleteLocalCluster(log, r.Client, cr); err != nil {
				return ctrl.Result{}, err
			}
			tracker.vzState = vzStateUninstallMC

		case vzStateUninstallMC:
			if err := r.deleteMCResources(log); err != nil {
				return ctrl.Result{}, err
			}
			tracker.vzState = vzStateUninstallComponents

		case vzStateUninstallComponents:
			log.Once("Uninstalling all Verrazzano components")
			res, err := r.uninstallComponents(log, cr, tracker)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = vzStateUninstallDone

		case vzStateUninstallDone:
			log.Once("Successfully uninstalled all Verrazzano components")
			tracker.vzState = vzStateUninstallEnd

		case vzStateUninstallEnd:
			done = true
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

// DeleteUninstallTracker deletes the Uninstall tracker for the Verrazzano resource
// This needs to be called when uninstall is completely done
func DeleteUninstallTracker(cr *installv1alpha1.Verrazzano) {
	key := getNSNKey(cr)
	_, ok := UninstallTrackerMap[key]
	if ok {
		delete(UninstallTrackerMap, key)
	}
}

// Delete multicluster related resources
func (r *Reconciler) deleteMCResources(log vzlog.VerrazzanoLogger) error {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: vzconst.VerrazzanoSystemNamespace,
		Name:      vzconst.MCAgentSecret,
	}
	// Get the MC secret
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		if errors.IsNotFound(err) {
			log.Once("Determined that this is not a managed cluster")
			return nil
		}
		return log.ErrorfNewErr("Failed to fetch the multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
	}
	log.Oncef("Deleting multicluster secret %s:%s", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret)
	if err := r.Delete(context.TODO(), &secret); err != nil {
		// Treat error as warning (don't fail uninstall)
		log.Oncef("Failed to delete multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
		return nil
	}

	log.Oncef("Deleting all VMC resources")
	vmcList := clustersapi.VerrazzanoManagedClusterList{}
	if err := r.List(context.TODO(), &vmcList, &client.ListOptions{}); err != nil {
		log.Errorf("Failed listing VMCs: %v", err)
		return err
	}
	for i, vmc := range vmcList.Items {
		if err := r.Delete(context.TODO(), &vmcList.Items[i]); err != nil {
			// Treat error as warning (don't fail uninstall)
			log.Oncef("Failed to delete VMC %s/%s, %v", vmc.Namespace, vmc.Name, err)
			return nil
		}
	}

	return nil
}
