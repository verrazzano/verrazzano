// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/basecontroller"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzv1alpha1v1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Specify the SPI interfaces that this controller implements
var _ controllerspi.Reconciler = Reconciler{}

type Reconciler struct {
	Client        client.Client
	Scheme        *runtime.Scheme
	ModuleClass   moduleapi.ModuleClassType
	DryRun        bool
	StatusUpdater *healthcheck.VerrazzanoStatusUpdater
}

// InitController start the  controller
func InitController(mgr ctrlruntime.Manager) error {
	controller := Reconciler{}

	// The config MUST contain at least the Reconciler.  Other spi interfaces are optional.
	config := basecontroller.ControllerConfig{
		Reconciler: &controller,
		Finalizer:  &controller,
		Watcher:    &controller,
	}
	baseController, err := basecontroller.CreateControllerAndAddItToManager(mgr, config)
	if err != nil {
		return err
	}

	// init other controller fields
	controller.Client = baseController.Client
	controller.Scheme = baseController.Scheme
	controller.StatusUpdater = healthcheck.NewStatusUpdater(mgr.GetClient())

	return nil
}

// GetReconcileObject returns the kind of object being reconciled
func (r Reconciler) GetReconcileObject() client.Object {
	return &vzv1alpha1v1beta1.Verrazzano{}
}
