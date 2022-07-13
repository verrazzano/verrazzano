// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	// vzStateUninstallCleanup is the state where the final cleanup is performed for a full uninstall
	vzStateUninstallCleanup uninstallState = "vzStateUninstallCleanup"

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
			tracker.vzState = vzStateUninstallCleanup

		case vzStateUninstallCleanup:
			err := r.uninstallCleanup(log)
			if err != nil {
				return ctrl.Result{}, err
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
	// Return if this is not MC or if there is an error
	if mc, err := r.isMC(log); err != nil || !mc {
		return err
	}

	log.Oncef("Deleting all VMC resources")
	vmcList := clustersapi.VerrazzanoManagedClusterList{}
	if err := r.List(context.TODO(), &vmcList, &client.ListOptions{}); err != nil {
		return log.ErrorfNewErr("Failed listing VMCs: %v", err)
	}
	for i, vmc := range vmcList.Items {
		if err := r.Delete(context.TODO(), &vmcList.Items[i]); err != nil {
			return log.ErrorfNewErr("Failed to delete VMC %s/%s, %v", vmc.Namespace, vmc.Name, err)
		}
	}

	// Delete VMC namespace only if there are no projects
	projects := vzappclusters.VerrazzanoProjectList{}
	if err := r.List(context.TODO(), &projects, &client.ListOptions{Namespace: vzconst.VerrazzanoMultiClusterNamespace}); err != nil {
		return log.ErrorfNewErr("Failed listing MC projects: %v", err)
	}
	if len(projects.Items) == 0 {
		log.Oncef("Deleting %s namespace", vzconst.VerrazzanoMultiClusterNamespace)
		if err := r.deleteNamespace(context.TODO(), log, vzconst.VerrazzanoMultiClusterNamespace); err != nil {
			return err
		}
	}

	// Delete secrets last. Don't delete MC agent secret until the end since it tells us this is MC install
	if err := r.deleteSecret(log, vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret); err != nil {
		return err
	}
	if err := r.deleteSecret(log, vzconst.VerrazzanoSystemNamespace, "verrazzano-cluster-elasticsearch"); err != nil {
		return err
	}
	if err := r.deleteSecret(log, vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) isMC(log vzlog.VerrazzanoLogger) (bool, error) {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: vzconst.VerrazzanoSystemNamespace,
		Name:      vzconst.MCAgentSecret,
	}

	// Get the MC agent secret and return if not found
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		if errors.IsNotFound(err) {
			log.Once("Determined that this is not a managed cluster")
			return false, nil
		}
		return false, log.ErrorfNewErr("Failed to fetch the multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
	}
	return true, nil
}

//uninstallCleanup Perform the final cleanup of shared resources, etc not tracked by individal component uninstalls
func (r *Reconciler) uninstallCleanup(log vzlog.VerrazzanoLogger) error {
	return r.deleteNamespaces(log)
}

func (r *Reconciler) deleteSecret(log vzlog.VerrazzanoLogger, namespace string, name string) error {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	log.Oncef("Deleting multicluster secret %s:%s", namespace, name)
	if err := r.Delete(context.TODO(), &secret); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return log.ErrorfNewErr("Failed to delete secret %s/%s, %v", namespace, name, err)
	}
	return nil
}

//deleteNamespaces Cleans up any namespaces shared by multiple components
func (r *Reconciler) deleteNamespaces(log vzlog.VerrazzanoLogger) error {
	return resource.Resource{
		Name:   promoperator.ComponentNamespace,
		Client: r.Client,
		Object: &corev1.Namespace{},
		Log:    log,
	}.RemoveFinalizersAndDelete()
}
