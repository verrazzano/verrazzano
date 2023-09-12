// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile/restart"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/verrazzano/custom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the Verrazzano CR.  This includes new installations, updates, upgrades, and partial uninstalls.
func (r Reconciler) Reconcile(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	zapLogForMetrics := zap.S().With(log.FieldController, "verrazzano")
	counterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileCounter)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		controllerCtx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}
	counterMetricObject.Inc()

	reconcileDurationMetricObject, err := metricsexporter.GetDurationMetric(metricsexporter.ReconcileDuration)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}
	reconcileDurationMetricObject.TimerStart()
	defer reconcileDurationMetricObject.TimerStop()
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	errorCounterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileError)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		errorCounterMetricObject.Inc()
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	// Do CR initialization
	if res := r.initVzResource(log, actualCR); res.ShouldRequeue() {
		return res
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not modified the version in the Verrazzano CR to match the BOM.
	if upgradePending, err := r.isUpgradeRequired(actualCR); upgradePending || err != nil {
		controllerCtx.Log.Oncef("Upgrade required before reconciling modules")
		if err != nil {
			errorCounterMetricObject.Inc()
		}
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		errorCounterMetricObject.Inc()
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	// Note: updating the VZ state to reconciling is done by the module shim to
	// avoid a long delay before the user sees any CR action.
	// See vzcomponent_status.go, UpdateVerrazzanoComponentStatus
	if r.isUpgrading(actualCR) {
		if err := r.updateStatusUpgrading(log, actualCR); err != nil {
			errorCounterMetricObject.Inc()
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Do global pre-work
	if res := r.preWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do the actual install, update, and or upgrade.
	if res := r.doWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		if res := r.updateStatusIfNeeded(log, actualCR); res.ShouldRequeue() {
			return res
		}
		return res
	}

	// Do global post-work
	if res := r.postWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// All done reconciling.  Add the completed condition to the status and set the state back to Ready.
	r.updateStatusInstallUpgradeComplete(actualCR)
	return result.NewResult()
}

// preWork does all the global pre-work for install and upgrade
func (r Reconciler) preWork(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// Pre-create the Verrazzano System namespace if it doesn't already exist, before kicking off the install job,
	// since it is needed for the subsequent step to syncLocalRegistration secret.
	istio := effectiveCR.Spec.Components.Istio
	if err := namespace.CreateVerrazzanoSystemNamespace(r.Client, istio != nil && istio.IsInjectionEnabled()); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Delete leftover MySQL backup job if we find one.
	if err := custom.CleanupMysqlBackupJob(log, r.Client); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if actualCR.Spec.Components.DNS != nil && actualCR.Spec.Components.DNS.OCI != nil {
		err := custom.DoesOCIDNSConfigSecretExist(r.Client, actualCR)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Sync the local cluster registration secret that allows the use of MC xyz resources on the
	// admin cluster without needing a VMC.
	if err := custom.SyncLocalRegistrationSecret(r.Client); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// create Rancher certs, etc.
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	custom.CreateRancherIngressAndCertCopies(componentCtx)

	return result.NewResult()
}

// doWork performs the verrazzano install, update, upgrade by creating, updating, or deleting modules
// Any combination of modules install, update, upgrade, and uninstall (delete) can be done at the same time.
// Return a requeue true until all modules are done doing work
func (r Reconciler) doWork(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// VZ components can be installed, updated, upgraded, or uninstalled independently
	// Process all the components and only requeue are the end, so that operations
	// (like uninstall) are not blocked by a different component's failure
	res1 := r.createOrUpdateModules(log, actualCR, effectiveCR)
	res2 := r.deleteModules(log, effectiveCR)

	// Requeue if any of the previous operations indicate a requeue is needed
	if res1.ShouldRequeue() || res2.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	// Return if modules are not done reconciling
	if !r.areModulesDoneReconciling(log, actualCR) {
		return result.NewResultShortRequeueDelay()
	}

	return result.NewResult()
}

// postWork does all the global post-work for install and upgrade
func (r Reconciler) postWork(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	if r.isUpgrading(actualCR) {
		return r.postUpgrade(log, actualCR)
	}
	return r.postInstall(log, actualCR)
}

// postInstallUpdate does all the global post-work for install and update
func (r Reconciler) postInstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := rancher.ConfigureAuthProviders(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := argocd.ConfigureKeycloakOIDC(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade ArgoCD configure OIDC: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// postUpgrade does all the global post-work for upgrade
func (r Reconciler) postUpgrade(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := rancher.ConfigureAuthProviders(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := argocd.ConfigureKeycloakOIDC(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade ArgoCD configure OIDC: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := mysql.PostUpgradeCleanup(log, componentCtx.Client()); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade MySQL cleanup: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if !r.areModulesDoneReconciling(log, actualCR) {
		log.Progress("Waiting for modules to be done in Verrazzano post-upgrade processing")
		return result.NewResultShortRequeueDelay()
	}

	if vzcr.IsApplicationOperatorEnabled(componentCtx.EffectiveCR()) && vzcr.IsIstioEnabled(componentCtx.EffectiveCR()) {
		err := restart.RestartApps(log, r.Client, actualCR.Generation)
		if err != nil {
			log.ErrorfThrottled("Failed Verrazzano post-upgrade application restarts: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	return result.NewResult()
}

// initVzResource initializes CR fields as needed.  This happens once when the CR is created
func (r Reconciler) initVzResource(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	// Add our finalizer if not already added
	if !vzstring.SliceContainsString(actualCR.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		actualCR.ObjectMeta.Finalizers = append(actualCR.ObjectMeta.Finalizers, finalizerName)
		if err := r.Client.Update(context.TODO(), actualCR); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Pre-populate the component status fields
	return r.initializeComponentStatus(log, actualCR)
}

// isUpgradeRequired Returns true if we detect that an upgrade is required but not (at least) in progress:
//   - if the Spec version IS NOT empty is less than the BOM version, an upgrade is required
//   - if the Spec version IS empty the Status version is less than the BOM, then an upgrade is required (upgrade of initial install scenario)
//
// If we return true here, it means we should stop reconciling until an upgrade has been requested
func (r Reconciler) isUpgradeRequired(actualCR *vzv1alpha1.Verrazzano) (bool, error) {
	if actualCR == nil {
		return false, fmt.Errorf("no Verrazzano CR provided")
	}
	bomVersion, err := validators.GetCurrentBomVersion()
	if err != nil {
		return false, err
	}

	if len(actualCR.Spec.Version) > 0 {
		specVersion, err := semver.NewSemVersion(actualCR.Spec.Version)
		if err != nil {
			return false, err
		}
		return specVersion.IsLessThan(bomVersion), nil
	}
	if len(actualCR.Status.Version) > 0 {
		statusVersion, err := semver.NewSemVersion(actualCR.Status.Version)
		if err != nil {
			return false, err
		}
		return statusVersion.IsLessThan(bomVersion), nil
	}
	return false, nil
}
