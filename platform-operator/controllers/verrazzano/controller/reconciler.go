// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"time"

	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzctrlcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/controller/custom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/restart"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile reconciles the Verrazzano CR.  This includes new installations, updates, upgrades, and partial uninstallations.
// Reconciliation is done by creating and updating Module CRs, one for each component that is enabled in the Verrazzano effective CR.
// If the Verrazzano component is disabled, then reconcile will uninstall that component by deleting the Module CR.  This code
// is idempotent and can be called any number of times from the controller-runtime.  If the Verrazzano CR gets modified while
// a life-cycle operation is already in progress, then those changes will take effect as soon as possible (when Reconcile is called)
//
// Reconciliation has 3 phases, pre-work, work, and post-work.  The global pre-work and post-work can block the entire controller,
// depending on what is being done.  The work phase, just creates,updates, and deletes the Module CR.
// Those operations are non-blocking, other than the time it takes to call the Kubernetes API server.
//
// NOTE: full uninstallations are done by the finalizer.go code
func (r Reconciler) Reconcile(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		// This is a fatal error which should never happen, don't requeue
		vzlog.DefaultLogger().Info("Failed to convert Unstructured object to Verrazzano CR")
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
		return result.NewResultRequeueDelay(1, 2, time.Minute)
	}

	// Intentionally ignore metrics errors so that it doesn't cause reconcile for fail
	counterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileCounter)
	if err != nil {
		log.ErrorfThrottled(err.Error())
	}
	counterMetricObject.Inc()

	// Intentionally ignore metrics errors so that it doesn't cause reconcile for fail
	errorCounterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileError)
	if err != nil {
		log.ErrorfThrottled(err.Error())
	}

	// Intentionally ignore metrics errors so that it doesn't cause reconcile for fail
	reconcileDurationMetricObject, err := metricsexporter.GetDurationMetric(metricsexporter.ReconcileDuration)
	if err != nil {
		log.ErrorfThrottled(err.Error())
	}

	reconcileDurationMetricObject.TimerStart()
	defer reconcileDurationMetricObject.TimerStop()

	res := r.doReconcile(log, controllerCtx, actualCR)
	if res.IsError() && metricsexporter.IsMetricError(res.GetError()) {
		errorCounterMetricObject.Inc()
	}

	// Requeue if reconcile is not done
	if res.ShouldRequeue() {
		return res
	}

	// Reconcile is complete
	controllerCtx.Log.Oncef("Successfully reconciled Verrazzano for generation %v", actualCR.Generation)
	metricsexporter.AnalyzeVerrazzanoResourceMetrics(log, *actualCR)
	return result.NewResult()
}

// doReconcile reconciles the Verrazzano CR.
func (r Reconciler) doReconcile(log vzlog.VerrazzanoLogger, controllerCtx controllerspi.ReconcileContext, actualCR *vzv1alpha1.Verrazzano) result.Result {
	// Do CR initialization
	if res := r.initVzResource(log, actualCR); res.ShouldRequeue() {
		return res
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not modified the version in the Verrazzano CR to match the BOM.
	if upgradeRequired, err := vzctrlcommon.IsUpgradeRequired(actualCR); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	} else if upgradeRequired {
		log.Oncef("Upgrade is required before reconciling %s", client.ObjectKeyFromObject(actualCR))
		return result.NewResultShortRequeueDelay()
	}

	effectiveCR, err := r.createEffectiveCR(actualCR)
	if err != nil {
		result.NewResultShortRequeueDelayWithError(err)
	}

	// Update the status if this is an upgrade. If this is not an upgrade,
	// then we don't know, at this point, if install, update, or partial uninstall is
	// needed, so the status update is deferred.  See vzstatus.updateStatusIfNeeded,
	// where the status is updated in those cases.
	if r.isUpgrading(actualCR) {
		if err := r.updateStatusUpgrading(log, actualCR); err != nil {
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
		return result.NewResultShortRequeueDelayWithError(err)
	}
	controllerCtx.Log.Oncef("Successfully reconciled Verrazzano for generation %v", actualCR.Generation)
	metricsexporter.AnalyzeVerrazzanoResourceMetrics(log, *actualCR)
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

// createEffectiveCR Creates an effective CR from the one passed in
func (r Reconciler) createEffectiveCR(actualCR *vzv1alpha1.Verrazzano) (*vzv1alpha1.Verrazzano, error) {
	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = actualCR.Status
	return effectiveCR, nil
}
