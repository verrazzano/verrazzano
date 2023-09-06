// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	modulelog "github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	vpovzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile/restart"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	moduleCatalog "github.com/verrazzano/verrazzano/platform-operator/experimental/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile reconciles the Verrazzano CR.  This includes new installations, updates, upgrades, and partial uninstalls.
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	log := spictx.Log

	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}

	// Do CR initialization
	if res := r.initVzResource(spictx.Log, actualCR); res.ShouldRequeue() {
		return res
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not modified the version in the Verrazzano CR to match the BOM.
	if upgradePending, err := r.isUpgradeRequired(actualCR); upgradePending || err != nil {
		spictx.Log.Oncef("Upgrade required before reconciling modules")
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	// Note: updating the VZ state to reconciling is done by the module shim to
	// avoid a long delay before the user sees any CR action.
	// See vzcomponent_status.go, UpdateVerrazzanoComponentStatus
	if r.isUpgrading(actualCR) {
		if err := r.updateStatusUpgrading(log, actualCR); err != nil {
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
	if err := r.createVerrazzanoSystemNamespace(context.TODO(), effectiveCR, log); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Sync the local cluster registration secret that allows the use of MC xyz resources on the
	// admin cluster without needing a VMC.
	if err := r.syncLocalRegistrationSecret(); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// create Rancher certs, etc.
	spiCtx, err := componentspi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	r.createRancherIngressAndCertCopies(spiCtx)

	return result.NewResult()
}

// doWork performs the verrazzano install, update, upgrade by creating, updating, or deleting modules
// Any combination of modules install, update, upgrade, and uninstall (delete) can be done at the same time.
// Return a requeue true until all modules are done doing work
func (r Reconciler) doWork(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// VZ components can be installed, updated, upgraded, or uninstalled independently
	// Process all the components and only requeue are the end, so that operations
	// (like uninstall) are not blocked by a different component's failure
	res1 := r.createOrUpdateModules(log, effectiveCR)
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
	} else {
		return r.postInstallUpdate(log, actualCR)
	}
}

// postInstallUpdate does all the global post-work for install and update
func (r Reconciler) postInstallUpdate(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	spiCtx, err := componentspi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := rancher.ConfigureAuthProviders(spiCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := argocd.ConfigureKeycloakOIDC(spiCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade ArgoCD configure OIDC: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// postUpgrade does all the global post-work for upgrade
func (r Reconciler) postUpgrade(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	spiCtx, err := componentspi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := restart.RestartComponents(vpovzlog.DefaultLogger(), config.GetInjectedSystemNamespaces(), spiCtx.ActualCR().Generation, &restart.OutdatedSidecarPodMatcher{}); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade restart components: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := rancher.ConfigureAuthProviders(spiCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade Rancher configure auth providers: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := argocd.ConfigureKeycloakOIDC(spiCtx); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade ArgoCD configure OIDC: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := restart.RestartComponents(vpovzlog.DefaultLogger(), config.GetInjectedSystemNamespaces(), spiCtx.ActualCR().Generation, &restart.OutdatedSidecarPodMatcher{}); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade restart components: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := mysql.PostUpgradeCleanup(vpovzlog.DefaultLogger(), spiCtx.Client()); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade MySQL cleanup: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if !r.areModulesDoneReconciling(log, actualCR) {
		log.Progress("Waiting for modules to be done in Verrazzano post-upgrade processing")
		return result.NewResultShortRequeueDelay()
	}

	if vzcr.IsApplicationOperatorEnabled(spiCtx.EffectiveCR()) && vzcr.IsIstioEnabled(spiCtx.EffectiveCR()) {
		err := restart.RestartApps(vpovzlog.DefaultLogger(), r.Client, actualCR.Generation)
		if err != nil {
			log.ErrorfThrottled("Failed Verrazzano post-upgrade application restarts: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	return result.NewResult()
}

// createOrUpdateModules creates or updates all the modules
func (r Reconciler) createOrUpdateModules(log vzlog.VerrazzanoLogger, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	catalog, err := moduleCatalog.NewCatalog(config.GetCatalogPath())
	if err != nil {
		log.ErrorfThrottled("Error loading module catalog: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		if !comp.ShouldUseModule() {
			continue
		}

		version := catalog.GetVersion(comp.Name())
		if version == nil {
			err = log.ErrorfThrottledNewErr("Failed to find version for module %s in the module catalog", comp.Name())
			return result.NewResultShortRequeueDelayWithError(err)
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: vzconst.VerrazzanoInstallNamespace,
			},
		}
		opResult, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			return r.mutateModule(log, effectiveCR, &module, comp, version.ToString())
		})
		log.Debugf("Module %s update result: %v", module.Name, opResult)
		if err != nil {
			if !errors.IsConflict(err) {
				log.ErrorfThrottled("Failed createOrUpdate module %s: %v", module.Name, err)
			}
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	return result.NewResult()
}

// mutateModule mutates the module for the create or update callback
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, effectiveCR *vzv1alpha1.Verrazzano, module *moduleapi.Module, comp componentspi.Component, moduleVersion string) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[vzconst.VerrazzanoCRNameAnnotation] = effectiveCR.Name
	module.Annotations[vzconst.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace

	if module.Labels == nil {
		module.Labels = make(map[string]string)
	}
	module.Labels[vzconst.VerrazzanoOwnerLabel] = string(effectiveCR.UID)

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()
	module.Spec.Version = moduleVersion

	return r.setModuleValues(log, effectiveCR, module, comp)
}

// initVzResource initializes CR fields as needed.  This happens once when the CR is created
func (r *Reconciler) initVzResource(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
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
