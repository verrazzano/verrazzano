// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzReconcile "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	moduleCatalog "github.com/verrazzano/verrazzano/platform-operator/experimental/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	if !vzReconcile.IsPreModuleWorkDone() {
		return result.NewResultShortRequeueDelay()
	}
	actualCR := &vzapi.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}

	// Wait for legacy verrazzano controller to initialize status
	if actualCR.Status.Components == nil {
		return result.NewResultShortRequeueDelay()
	}

	// Get effective CR and set the effective CR status with the actual status
	// Note that the reconciler code only udpdate the status, which is why the effective
	// CR is passed.  If was ever to update the spec, then the actual CR would need to be used.
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	log := vzlog.DefaultLogger()

	// VZ components can be installed, updated, upgraded, or uninstalled independently
	// Process all the components and only requeue are the end, so that operations
	// (like uninstall) are not blocked by a different component's failure
	res1 := r.deleteModules(log, effectiveCR)
	res2 := r.reconcileModules(log, effectiveCR)

	// Requeue if any of the previous operations indicate a requeue is needed
	if res1.ShouldRequeue() || res2.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	vzReconcile.SetModuleCreateOrUpdateStarted(false)
	vzReconcile.SetModuleCreateOrUpdateDone(true)

	// All the modules have been reconciled and are ready
	return result.NewResult()
}

// reconcileModules creates or updates all the modules
func (r Reconciler) reconcileModules(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	// if modules are not reconciling yet, then createOrUpdate them
	if !vzReconcile.IsModuleCreateOrUpdateStarted() {
		catalog, err := moduleCatalog.NewCatalog(config.GetCatalogPath())
		if err != nil {
			log.ErrorfThrottled("Error loading module catalog: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}

		// Create or Update a Module for each enabled resource
		for _, comp := range registry.GetComponents() {
			if err = r.mutateModule(log, effectiveCR, comp, catalog.GetVersion(comp.Name())); err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
		}
		vzReconcile.SetModuleCreateOrUpdateStarted(true)
		return result.NewResultShortRequeueDelay()
	}

	// if modules are reconciling see if they are all ready
	if !vzReconcile.IsModuleCreateOrUpdateDone() {
		for _, comp := range registry.GetComponents() {
			ready, err := r.modulesReady(log, effectiveCR, comp)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
			if !ready {
				return result.NewResultShortRequeueDelay()
			}
		}
		vzReconcile.SetModuleCreateOrUpdateDone(true)
	}
	return result.NewResult()
}

// mutateModule mutates the module for the create or update callback
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, comp componentspi.Component, moduleVersion *semver.SemVersion) error {
	if !comp.IsEnabled(effectiveCR) {
		return nil
	}
	if !comp.ShouldUseModule() {
		return nil
	}

	if moduleVersion == nil {
		return log.ErrorfThrottledNewErr("Failed to find version for module %s in the module catalog", comp.Name())
	}

	module := moduleapi.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      comp.Name(),
			Namespace: vzconst.VerrazzanoInstallNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
		if module.Annotations == nil {
			module.Annotations = make(map[string]string)
		}
		module.Annotations[vzconst.VerrazzanoCRNameAnnotation] = effectiveCR.Name
		module.Annotations[vzconst.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace

		module.Spec.ModuleName = module.Name
		module.Spec.TargetNamespace = comp.Namespace()
		module.Spec.Version = moduleVersion.ToString()

		return r.setModuleValues(log, effectiveCR, &module, comp)
	})
	if err != nil {
		if !errors.IsConflict(err) {
			log.ErrorfThrottled("Failed createOrUpdate module %s: %v", module.Name, err)
		}
		return err
	}
	return nil
}

func (r Reconciler) modulesReady(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, comp componentspi.Component) (bool, error) {
	if !comp.IsEnabled(effectiveCR) {
		return true, nil
	}
	if !comp.ShouldUseModule() {
		return true, nil
	}

	readyCondition := false
	generationEqual := false
	versionEqual := false
	module := moduleapi.Module{}
	nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: comp.Name()}
	err := r.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{})
	if err != nil {
		log.ErrorfThrottled("Failed to get Module %s, retrying: %v", comp.Name(), err)
		return false, err
	}

	if module.Status.Conditions[len(module.Status.Conditions)-1].Type == moduleapi.ModuleConditionReady {
		readyCondition = true
	}
	if module.Status.LastSuccessfulGeneration == module.Generation {
		generationEqual = true
	}
	if module.Status.LastSuccessfulVersion == module.Spec.Version {
		versionEqual = true
	}
	return readyCondition && generationEqual && versionEqual, nil
}
