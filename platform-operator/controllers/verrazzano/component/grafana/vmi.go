// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// updateFunc mutates the VMI struct and ensures the Grafana component is configured properly
var updateFunc common.VMIMutateFunc = func(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	vmi.Spec.Grafana = newGrafana(ctx.EffectiveCR(), storage, existingVMI)
	return nil
}

// newGrafana creates a Grafana struct and populates it using existing config or default values
func newGrafana(cr *vzapi.Verrazzano, storage *common.ResourceRequestValues, existingVMI *vmov1.VerrazzanoMonitoringInstance) vmov1.Grafana {
	grafanaSpec := cr.Spec.Components.Grafana
	if grafanaSpec == nil {
		return vmov1.Grafana{}
	}
	grafana := vmov1.Grafana{
		Enabled:              grafanaSpec.Enabled != nil && *grafanaSpec.Enabled,
		DashboardsConfigMap:  "system-dashboards",
		DatasourcesConfigMap: "vmi-system-datasource",
		Resources: vmov1.Resources{
			RequestMemory: "48Mi",
		},
		Storage: vmov1.Storage{},
	}
	setStorageSize(storage, &grafana.Storage)
	if existingVMI != nil {
		grafana.Storage = existingVMI.Spec.Grafana.Storage
	}
	return grafana
}

// setStorageSize copies or defaults the storage size
func setStorageSize(storage *common.ResourceRequestValues, storageObject *vmov1.Storage) {
	if storage == nil {
		storageObject.Size = "50Gi"
	} else {
		storageObject.Size = storage.Storage
	}
}
