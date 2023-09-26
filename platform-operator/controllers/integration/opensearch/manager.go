// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/basecontroller"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Specify the SPI interfaces that this controller implements
var _ controllerspi.Reconciler = Reconciler{}

type Reconciler struct {
	Client      client.Client
	log         vzlog.VerrazzanoLogger
	Scheme      *runtime.Scheme
	ModuleClass moduleapi.ModuleClassType
	DryRun      bool
}

// InitController start the  controller
func InitController(mgr ctrlruntime.Manager) error {
	controller := Reconciler{}

	// The config MUST contain at least the Reconciler.  Other spi interfaces are optional.
	config := basecontroller.ControllerConfig{
		Reconciler:  &controller,
		EventFilter: &controller,
	}
	baseController, err := basecontroller.CreateControllerAndAddItToManager(mgr, config)
	if err != nil {
		return err
	}

	// init other controller fields
	controller.Client = baseController.Client
	controller.Scheme = baseController.Scheme
	return nil
}

// GetReconcileObject returns the kind of object being reconciled
func (r Reconciler) GetReconcileObject() client.Object {
	return &corev1.ConfigMap{}
}

// HandlePredicateEvent returns true if this is the OpenSearch integration operator configmap.
func (r Reconciler) HandlePredicateEvent(cli client.Client, object client.Object) bool {
	return object.GetNamespace() == constants.VerrazzanoInstallNamespace &&
		object.GetName() == constants.OpenSearchIntegrationConfigMapName
}
