// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
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
	res1 := r.createOrUpdateModules(log, effectiveCR)
	res2 := r.deleteModules(log, effectiveCR)

	// Requeue if any of the previous operations indicate a requeue is needed
	if res1.ShouldRequeue() || res2.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	// All the modules have been reconciled and are ready
	return result.NewResult()
}

// createOrUpdateModules creates or updates all the modules
func (r Reconciler) createOrUpdateModules(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	semver, err := validators.GetCurrentBomVersion()
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(fmt.Errorf("Failed to get BOM version: %v", err))
	}

	version := semver.ToString()

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		if !comp.ShouldUseModule() {
			continue
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: constants.VerrazzanoInstallNamespace,
			},
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			// TODO For now have the module version match the VZ version
			return r.mutateModule(log, effectiveCR, &module, comp, version)
		})
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
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp componentspi.Component, moduleVersion string) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[constants.VerrazzanoCRNameAnnotation] = effectiveCR.Name
	module.Annotations[constants.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()
	module.Spec.Version = moduleVersion

	var err error
	module.Spec.Values, err = comp.GetModuleSpec(effectiveCR)
	if err != nil {
		return err
	}

	return r.setModuleValues(log, effectiveCR, module, comp)
}
