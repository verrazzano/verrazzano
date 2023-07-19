// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

// initializedSet is needed to keep track of which Verrazzano CRs have been initialized
var initializedSet = make(map[string]bool)

// Set to true during unit testing
var unitTesting bool

// Reconcile reconciles the Module CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	actualCR := &vzapi.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
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

	// Temp disable, this is done by the legacy Verrazzano controller
	//res := r.initStatus(log, effectiveCR)
	//if res.ShouldRequeue() {
	//	return res
	//}

	err = r.createOrUpdateModules(effectiveCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	ready, err := r.updateStatusForComponents(log, effectiveCR)
	if err != nil || !ready {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// All the modules have been reconciled and are ready
	return result.NewResult()
}

// createOrUpdateModules creates or updates all the modules
func (r Reconciler) createOrUpdateModules(effectiveCR *vzapi.Verrazzano) error {
	semver, err := validators.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	version := semver.ToString()

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}

		createOrUpdate, err := r.shouldCreateOrUpdateModule(effectiveCR, comp)
		if err != nil {
			return err
		}
		if !createOrUpdate {
			continue
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: constants.VerrazzanoInstallNamespace,
			},
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			return mutateModule(effectiveCR, &module, comp, version)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// mutateModule mutates the module for the create or update callback
func mutateModule(effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp spi.Component, version string) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[constants.VerrazzanoCRNameAnnotation] = effectiveCR.Name
	module.Annotations[constants.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace
	module.Annotations[constants.VerrazzanoObservedGeneration] = strconv.FormatInt(effectiveCR.Generation, 10)

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()

	// TODO For now have the module version match the VZ version
	module.Spec.Version = version
	return nil
}

// shouldCreateOrUpdateModule returns true if the Module should be created or updated
func (r Reconciler) shouldCreateOrUpdateModule(effectiveCR *vzapi.Verrazzano, comp spi.Component) (bool, error) {
	if !comp.ShouldUseModule() {
		return false, nil
	}
	if !comp.IsEnabled(effectiveCR) {
		return false, nil
	}

	// get the module
	module := &moduleapi.Module{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: comp.Name()}, module); err != nil {
		if errors.IsNotFound(err) {
			// module doesn't exist, need to create it
			return true, nil
		}
		return false, err
	}

	// if module doesn't have the current VZ generation then return true
	if module.Annotations != nil {
		gen := module.Annotations[constants.VerrazzanoObservedGeneration]
		return gen != strconv.FormatInt(effectiveCR.Generation, 10), nil
	}

	return true, nil
}

// Temp disable since legacy Verrazzano controller does this
//// Init status fields
//func (r Reconciler) initStatus(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
//	// Init the state to Ready if this CR has never been processed
//	// Always requeue to update cache, ignore error since requeue anyway
//	if len(effectiveCR.Status.State) == 0 {
//		effectiveCR.Status.State = vzapi.VzStateReconciling
//		r.updateStatus(log, effectiveCR)
//		return result.NewResultShortRequeueDelay()
//	}
//
//	// Check if init done for this resource
//	_, ok := initializedSet[effectiveCR.Name]
//	if ok {
//		return result.NewResult()
//	}
//
//	// Pre-populate the component status fields
//	res := r.initializeComponentStatus(log, effectiveCR)
//	if res.ShouldRequeue() {
//		return res
//	}
//
//	// Update the map indicating the resource has been initialized
//	initializedSet[effectiveCR.Name] = true
//	return result.NewResult()
//}

// updateStatusForComponents updates the vz CR status for the components based on the module status
// return true if all components are ready
func (r Reconciler) updateStatusForComponents(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) (bool, error) {
	var readyCount int
	var moduleCount int

	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		moduleCount++

		// get the module
		module := &moduleapi.Module{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: comp.Name()}, module); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.ErrorfThrottled("Failed getting Module %s: %v", comp.Name(), err)
			continue
		}
		// Set the effectiveCR status from the module status
		compStatus := r.loadModuleStatusIntoComponentStatus(effectiveCR, comp.Name(), module)
		if compStatus.State == vzapi.CompStateReady {
			readyCount++
		}
	}

	vzReady := moduleCount == readyCount
	if vzReady {
		effectiveCR.Status.State = vzapi.VzStateReady
	}

	// Update the status.  If it didn't change then the Kubernetes API server will not be called
	err := r.Client.Status().Update(context.TODO(), effectiveCR)
	if err != nil {
		return false, err
	}

	// return true if all modules are ready
	return vzReady, nil
}
