package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	moduleCatalog "github.com/verrazzano/verrazzano/platform-operator/experimental/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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

// deleteModules deletes all the modules, optionally only deleting ones that disabled
func (r Reconciler) deleteModules(log vzlog.VerrazzanoLogger, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	var reterr error
	var deletedCount int
	var moduleCount int

	// If deletion timestamp is non-zero then the VZ CR got deleted
	fullUninstall := !effectiveCR.GetDeletionTimestamp().IsZero()

	// Delete all modules.  Loop through all the modules once even if error occurs.
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}

		// If not full uninstall then only disabled components should be uninstalled
		if !fullUninstall && comp.IsEnabled(effectiveCR) {
			continue
		}

		// Check if the module exists before trying to delete the other related resources
		module := moduleapi.Module{}
		nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: comp.Name()}
		err := r.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Progressf("Failed to get Module %s, retrying: %v", comp.Name(), err)
			return result.NewResultShortRequeueDelayIfError(err)
		}

		moduleCount++

		// Delete all the configuration secrets that were referenced by the module
		res := r.deleteConfigSecrets(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		// Delete all the configuration configmaps that were referenced by the module
		res = r.deleteConfigMaps(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		// Delete the Module
		err = r.Client.Delete(context.TODO(), &module, &client.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				deletedCount++
				continue
			}
			if !errors.IsConflict(err) {
				log.Progressf("Failed to delete Module %s, retrying: %v", comp.Name(), err)
			}
			reterr = err
			continue
		}
	}
	if deletedCount != moduleCount {
		return result.NewResultShortRequeueDelay()
	}

	// return last error found so that we retry
	if reterr != nil {
		return result.NewResultShortRequeueDelayWithError(reterr)
	}

	// All modules have been deleted and the Module CRs are gone
	return result.NewResult()
}
