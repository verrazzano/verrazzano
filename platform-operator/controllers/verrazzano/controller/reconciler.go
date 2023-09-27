// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	custom2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/controller/custom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/restart"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the Verrazzano CR.  This includes new installations, updates, upgrades, and partial uninstalls.
// NOTE: full uninstalls are done by the finalizer.go code
func (r Reconciler) Reconcile(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Initialize metrics
	zapLogForMetrics := zap.S().With(log.FieldController, "verrazzano")
	counterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileCounter)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}
	counterMetricObject.Inc()
	errorCounterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileError)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}

	reconcileDurationMetricObject, err := metricsexporter.GetDurationMetric(metricsexporter.ReconcileDuration)
	if err != nil {
		zapLogForMetrics.Error(err)
		return result.NewResult()
	}
	reconcileDurationMetricObject.TimerStart()
	defer reconcileDurationMetricObject.TimerStop()
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		controllerCtx.Log.ErrorfThrottled(err.Error())
		if metricsexporter.IsMetricError(err) {
			errorCounterMetricObject.Inc()
		}
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
		if metricsexporter.IsMetricError(err) {
			errorCounterMetricObject.Inc()
		}
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	// Do CR initialization
	if res := r.initVzResource(log, actualCR); res.ShouldRequeue() {
		return res
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not modified the version in the Verrazzano CR to match the BOM.
	if upgradePending, err := r.isUpgradeRequired(actualCR); upgradePending || err != nil {
		if err != nil && metricsexporter.IsMetricError(err) {
			errorCounterMetricObject.Inc()
		}
		controllerCtx.Log.Oncef("Upgrade required before reconciling modules")
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		if metricsexporter.IsMetricError(err) {
			errorCounterMetricObject.Inc()
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	// Update the status if this is an upgrade. If this is not an upgrade,
	// then we don't know, at this point, if install, update, or partial uninstall is
	// needed, so the status update is deferred.  See vzstatus.updateStatusIfNeeded,
	// where the status is updated in those cases.
	if r.isUpgrading(actualCR) {
		if err := r.updateStatusUpgrading(log, actualCR); err != nil {
			if metricsexporter.IsMetricError(err) {
				errorCounterMetricObject.Inc()
			}
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	controllerCtx.Log.Oncef("Started reconciling Verrazzano for generation %v", actualCR.Generation)

	// Do global pre-work
	if res := r.preWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do the actual install, update, and or upgrade.
	if res := r.doWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do global post-work
	if res := r.postWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// All done reconciling.  Add the completed condition to the status and set the state back to Ready.
	if err := r.updateStatusInstallUpgradeComplete(actualCR); err != nil {
		if metricsexporter.IsMetricError(err) {
			errorCounterMetricObject.Inc()
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	controllerCtx.Log.Oncef("Successfully reconciled Verrazzano for generation %v", actualCR.Generation)
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
	if err := custom2.CleanupMysqlBackupJob(log, r.Client); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if actualCR.Spec.Components.DNS != nil && actualCR.Spec.Components.DNS.OCI != nil {
		err := custom2.DoesOCIDNSConfigSecretExist(r.Client, actualCR)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Sync the local cluster registration secret that allows the use of MC xyz resources on the
	// admin cluster without needing a VMC.
	if err := custom2.SyncLocalRegistrationSecret(r.Client); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// create Rancher certs, etc.
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	custom2.CreateRancherIngressAndCertCopies(componentCtx)

	return result.NewResult()
}

// doWork performs the verrazzano install, update, upgrade, or partial uninstall by creating, updating, or deleting modules
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

	log.Once("Global post-install: configuring auth providers, if needed")
	if err := rancher.ConfigureAuthProviders(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	log.Once("Global post-install: configuring ArgoCD OIDC, if needed")
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

	log.Once("Global post-upgrade: configuring auth providers, if needed")
	if err := rancher.ConfigureAuthProviders(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	log.Once("Global post-upgrade: configuring ArgoCD OIDC, if needed")
	if err := argocd.ConfigureKeycloakOIDC(componentCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade ArgoCD configure OIDC: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Make sure namespaces get updated with Istio Enabled
	common.CreateAndLabelNamespaces(componentCtx)

	log.Once("Global post-upgrade: restarting all components that have an old Istio proxy sidecar")
	if err := restart.RestartComponents(log, config.GetInjectedSystemNamespaces(), componentCtx.ActualCR().Generation, &restart.OutdatedSidecarPodMatcher{}); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade restart components: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	log.Once("Global post-upgrade: doing MySQL post-upgrade cleanup")
	if err := mysql.PostUpgradeCleanup(log, componentCtx.Client()); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade MySQL cleanup: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if !r.areModulesDoneReconciling(log, actualCR) {
		log.Progress("Global post-upgrade: waiting for modules to be done")
		return result.NewResultShortRequeueDelay()
	}

	if vzcr.IsApplicationOperatorEnabled(componentCtx.EffectiveCR()) && vzcr.IsIstioEnabled(componentCtx.EffectiveCR()) {
		log.Once("Global post-upgrade: restarting all applications that have an old Istio proxy sidecar")
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
