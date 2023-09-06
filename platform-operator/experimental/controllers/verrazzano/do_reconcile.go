// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile/restart"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// preWork does all the global pre-work for install and upgrade
func (r Reconciler) preWork(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// Pre-create the Verrazzano System namespace if it doesn't already exist, before kicking off the install job,
	// since it is needed for the subsequent step to syncLocalRegistration secret.
	if err := r.createVerrazzanoSystemNamespace(context.TODO(), effectiveCR, log); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Delete leftover MySQL backup job if we find one.
	if err := r.cleanupMysqlBackupJob(log); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if actualCR.Spec.Components.DNS != nil && actualCR.Spec.Components.DNS.OCI != nil {
		err := r.doesOCIDNSConfigSecretExist(actualCR)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Sync the local cluster registration secret that allows the use of MC xyz resources on the
	// admin cluster without needing a VMC.
	if err := r.syncLocalRegistrationSecret(); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// create Rancher certs, etc.
	spiCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
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
	}
	return r.postInstall(log, actualCR)
}

// postInstallUpdate does all the global post-work for install and update
func (r Reconciler) postInstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	spiCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
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
	spiCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
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

	if err := restart.RestartComponents(log, config.GetInjectedSystemNamespaces(), spiCtx.ActualCR().Generation, &restart.OutdatedSidecarPodMatcher{}); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade restart components: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := mysql.PostUpgradeCleanup(log, spiCtx.Client()); err != nil {
		log.ErrorfThrottled("Failed Verrazzano post-upgrade MySQL cleanup: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if !r.areModulesDoneReconciling(log, actualCR) {
		log.Progress("Waiting for modules to be done in Verrazzano post-upgrade processing")
		return result.NewResultShortRequeueDelay()
	}

	if vzcr.IsApplicationOperatorEnabled(spiCtx.EffectiveCR()) && vzcr.IsIstioEnabled(spiCtx.EffectiveCR()) {
		err := restart.RestartApps(log, r.Client, actualCR.Generation)
		if err != nil {
			log.ErrorfThrottled("Failed Verrazzano post-upgrade application restarts: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	return result.NewResult()
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

// doesOCIDNSConfigSecretExist returns true if the DNS secret exists
func (r *Reconciler) doesOCIDNSConfigSecretExist(vz *vzv1alpha1.Verrazzano) error {
	// ensure the secret exists before proceeding
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: vz.Spec.Components.DNS.OCI.OCIConfigSecret, Namespace: vzconst.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		return err
	}
	return nil
}
