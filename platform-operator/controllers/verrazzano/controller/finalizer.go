// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/controller/custom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const finalizerName = "install.verrazzano.io"

// GetName returns the name of the finalizer
func (r Reconciler) GetName() string {
	return finalizerName
}

// PreRemoveFinalizer is called when the resource is being deleted, before the finalizer is removed.
// This code will do a full Verrazzano uninstall by deleting all the Module CRs. This code
// idempotent and can be called any number of times from the controller-runtime.  It doesn't matter if the Verrazzano CR
// gets modified while uninstall already in progress, because the Reconcile method will not be called again for that particular
// Verrazzano CR once the CR deletion timestamp is set.
//
// Uninstall has 3 phases, pre-work, work, and post-work.  The global pre-work and post-work can block the entire controller,
// depending on what is being done.  The work phase deletes the Module CRs then checks if they are gone.
// Those delete operations are non-blocking, other than the time it takes to call the Kubernetes API server.
//
// Once the Modules are gone and the post-work is done, then the finalizer is removed, causing the Verrazzano CR to be deleted from etcd.
func (r Reconciler) PreRemoveFinalizer(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		controllerCtx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	r.updateStatusUninstalling(log, actualCR)

	// Get effective CR and set the effective CR status with the actual status
	// Note that the reconciler code only udpdate the status, which is why the effective
	// CR is passed.  If was ever to update the spec, then the actual CR would need to be used.
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	controllerCtx.Log.Oncef("Started uninstalling Verrazzano")

	// Do global pre-work
	if res := r.preUninstall(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do the actual install, update, and or upgrade.
	if res := r.doUninstall(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do global post-work
	if res := r.postUninstall(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// All done reconciling.  Add the completed condition to the status and set the state back to Ready.
	if err := r.updateStatusUninstallComplete(actualCR); err != nil {
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// All install related resources have been deleted, delete the finalizer so that the Verrazzano
	// resource can get removed from etcd.
	log.Oncef("Removing finalizer %s", finalizerName)
	actualCR.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(actualCR.ObjectMeta.Finalizers, finalizerName)
	if err := r.Client.Update(context.TODO(), actualCR); err != nil {
		return result.NewResultShortRequeueDelayIfError(err)
	}

	controllerCtx.Log.Oncef("Successfully uninstalled Verrazzano")
	return result.NewResult()
}

// PostRemoveFinalizer is called after the finalizer is successfully removed.
// This method does garbage collection and other tasks that can never return an error
func (r Reconciler) PostRemoveFinalizer(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) {
	// Delete the tracker used for this CR
	//statemachine.DeleteTracker(u)
}

// preUninstall does all the global preUninstall
func (r Reconciler) preUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	if res := custom.PreUninstallRancher(r.Client, log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	if res := r.preUninstallMC(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	return result.NewResult()
}

// doUninstall performs the verrazzano uninstall by deleting modules
// Return a requeue true until all modules are gone
func (r Reconciler) doUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// Delete modules that are enabled and update status
	// Don't block status update of component if delete failed
	res := r.deleteModules(log, effectiveCR)
	if res.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	return result.NewResult()
}

// postUninstall does all the global postUninstall
func (r Reconciler) postUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if res := r.postUninstallCleanup(log, componentCtx); res.ShouldRequeue() {
		return res
	}
	return result.NewResult()
}

// preUninstallMC does MC pre-uninstall
func (r Reconciler) preUninstallMC(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := custom.DeleteMCResources(componentCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// uninstallCleanup Perform the final cleanup of shared resources, etc not tracked by individual component uninstalls
func (r Reconciler) postUninstallCleanup(log vzlog.VerrazzanoLogger, componentCtx componentspi.ComponentContext) result.Result {
	rancherProvisioned, err := rancher.IsClusterProvisionedByRancher()
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	log.Once("Global post-uninstall: deleting Istio CA Root Cert")
	if err := custom.DeleteIstioCARootCert(componentCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	log.Once("Global post-uninstall: node exporter cleanup")
	if err := custom.NodeExporterCleanup(componentCtx.Client(), componentCtx.Log()); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Run Rancher Post Uninstall explicitly to delete any remaining Rancher resources; this may be needed in case
	// the uninstall was interrupted during uninstall, or if the cluster is a managed cluster where Rancher is not
	// installed explicitly.
	if !rancherProvisioned {
		log.Once("Global post-uninstall: running Rancher cleanup")
		if err := custom.RunRancherPostUninstall(componentCtx); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	log.Once("Global post-uninstall: deleting namespaces")
	return custom.DeleteNamespaces(componentCtx, rancherProvisioned)
}
