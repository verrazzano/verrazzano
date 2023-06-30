// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var localbom *bom.Bom

// initializedSet is needed to keep track of which Verrazzano CRs have been initialized
var initializedSet = make(map[string]bool)

// systemNamespaceLabels the verrazzano-system namespace labels required
var systemNamespaceLabels = map[string]string{
	"istio-injection":         "enabled",
	"verrazzano.io/namespace": vzconst.VerrazzanoSystemNamespace,
}

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

	log := vzlog.DefaultLogger()
	res := r.initReconcile(log, actualCR)
	if res.ShouldRequeue() {
		return res
	}

	// Get effective CR
	vzcr, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Create a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(vzcr) {
			continue
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: constants.VerrazzanoInstallNamespace,
			},
		}
		_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			return mutateModule(vzcr.Name, vzcr.Namespace, &module, comp)
		})
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	return result.NewResult()
}

func mutateModule(vzName string, vzNamespace string, module *moduleapi.Module, comp spi.Component) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[constants.VerrazzanoCRNameAnnotation] = vzName
	module.Annotations[constants.VerrazzanoCRNamespaceAnnotation] = vzNamespace

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()
	return nil
}

func (r *Reconciler) getBOM() (*bom.Bom, error) {
	if localbom == nil {
		newbom, err := bom.NewBom(config.GetDefaultBOMFilePath())
		if err != nil {
			return nil, err
		}
		localbom = &newbom
	}
	return localbom, nil
}

// initForVzResource will do initialization for the given Verrazzano resource.
// Clean up old resources from a 1.0 release where jobs, etc were in the default namespace
// Add a watch for each Verrazzano resource
func (r *Reconciler) initForVzResource(vz *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) result.Result {
	// Add our finalizer if not already added
	if !vzstring.SliceContainsString(vz.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		vz.ObjectMeta.Finalizers = append(vz.ObjectMeta.Finalizers, finalizerName)
		if err := r.Client.Update(context.TODO(), vz); err != nil {
			return result.NewResultShortRequeueDelay()
		}
	}

	if unitTesting {
		return result.NewResult()
	}

	// Check if init done for this resource
	_, ok := initializedSet[vz.Name]
	if ok {
		return result.NewResult()
	}

	// Update the map indicating the resource is being watched
	initializedSet[vz.Name] = true
	return result.NewResultShortRequeueDelay()
}

func (r Reconciler) initReconcile(log vzlog.VerrazzanoLogger, actualCR *vzapi.Verrazzano) result.Result {
	// Init the state to Ready if this CR has never been processed
	// Always requeue to update cache, ignore error since requeue anyway
	if len(actualCR.Status.State) == 0 {
		r.updateVzState(log, actualCR, vzapi.VzStateReady)
		return result.NewResultShortRequeueDelay()
	}

	// Initialize once for this Verrazzano resource when the operator starts
	res := r.initForVzResource(actualCR, log)
	if res.ShouldRequeue() {
		return res
	}

	// Pre-populate the component status fields
	res = r.initializeComponentStatus(log, actualCR)
	if res.ShouldRequeue() {
		return res
	}
	return result.NewResult()
}
