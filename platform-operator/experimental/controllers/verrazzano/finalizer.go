// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const finalizerName = "install.verrazzano.io"

// old node-exporter constants replaced with prometheus-operator node-exporter
const (
	monitoringNamespace = "monitoring"
	nodeExporterName    = "node-exporter"
	mcElasticSearchScrt = "verrazzano-cluster-elasticsearch"
	istioRootCertName   = "istio-ca-root-cert"
)

// sharedNamespaces The set of namespaces shared by multiple components; managed separately apart from individual components
var sharedNamespaces = []string{
	vzconst.VerrazzanoMonitoringNamespace,
	constants.CertManagerNamespace,
	constants.VerrazzanoSystemNamespace,
	vzconst.KeycloakNamespace,
	monitoringNamespace,
}

// GetName returns the name of the finalizer
func (r Reconciler) GetName() string {
	return finalizerName
}

// PreRemoveFinalizer is called when the resource is being deleted, before the finalizer
// is removed.  Use this method to delete Kubernetes resources, etc.
func (r Reconciler) PreRemoveFinalizer(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
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

	// Always requeue, the legacy verrazzano controller will delete the finalizer and the VZ CR will go away.
	return result.NewResult()
}

// PostRemoveFinalizer is called after the finalizer is successfully removed.
// This method does garbage collection and other tasks that can never return an error
func (r Reconciler) PostRemoveFinalizer(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) {
	// Delete the tracker used for this CR
	//statemachine.DeleteTracker(u)
}
