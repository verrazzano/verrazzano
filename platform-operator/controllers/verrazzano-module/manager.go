// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package module

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/basecontroller"
	spi "github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Specify the SPI interfaces that this controller implements
var _ spi.Reconciler = Reconciler{}

type Reconciler struct {
	Client      client.Client
	Scheme      *runtime.Scheme
	HandlerInfo handlerspi.ModuleHandlerInfo
	ModuleClass moduleapi.ModuleClassType
}

// InitController start the  controller
func InitController(mgr ctrlruntime.Manager, handlerInfo handlerspi.ModuleHandlerInfo, class moduleapi.ModuleClassType) error {
	controller := Reconciler{}

	// The config MUST contain at least the Reconciler.  Other spi interfaces are optional.
	config := basecontroller.ControllerConfig{
		Reconciler:  &controller,
		Finalizer:   &controller,
		EventFilter: &controller,
	}
	baseController, err := basecontroller.CreateControllerAndAddItToManager(mgr, config)
	if err != nil {
		return err
	}

	// init other controller fields
	controller.Client = baseController.Client
	controller.Scheme = baseController.Scheme
	controller.HandlerInfo = handlerInfo
	controller.ModuleClass = class
	return nil
}

// GetReconcileObject returns the kind of object being reconciled
func (r Reconciler) GetReconcileObject() client.Object {
	return &moduleapi.Module{}
}

func (r Reconciler) HandlePredicateEvent(cli client.Client, object client.Object) bool {
	mlc := moduleapi.Module{}
	objectkey := client.ObjectKeyFromObject(object)
	if err := cli.Get(context.TODO(), objectkey, &mlc); err != nil {
		return false
	}
	return mlc.Spec.ModuleName == string(r.ModuleClass)
}
