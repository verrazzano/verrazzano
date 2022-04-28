// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const (
	nodeExporter = "node-exporter"
)

// updateFunc is passed into CreateOrUpdateVMI to create the necessary VMI resources
func updateFunc(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	vmi.Spec.Grafana = newGrafana(ctx.EffectiveCR(), storage, existingVMI)
	vmi.Spec.Prometheus = newPrometheus(ctx.EffectiveCR(), storage, existingVMI)
	return nil
}

func newGrafana(cr *vzapi.Verrazzano, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Grafana {
	if cr.Spec.Components.Grafana == nil {
		return vmov1.Grafana{}
	}
	grafanaValues := cr.Spec.Components.Grafana
	grafana := vmov1.Grafana{
		Enabled:              grafanaValues.Enabled != nil && *grafanaValues.Enabled,
		DashboardsConfigMap:  "system-dashboards",
		DatasourcesConfigMap: "vmi-system-datasource",
		Resources: vmov1.Resources{
			RequestMemory: "48Mi",
		},
		Storage: vmov1.Storage{},
	}
	setStorageSize(storage, &grafana.Storage)
	if vmi != nil {
		grafana.Storage = vmi.Spec.Grafana.Storage
	}

	return grafana
}

func newPrometheus(cr *vzapi.Verrazzano, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Prometheus {
	if cr.Spec.Components.Prometheus == nil {
		return vmov1.Prometheus{}
	}
	prometheusValues := cr.Spec.Components.Prometheus
	prometheus := vmov1.Prometheus{
		Enabled: prometheusValues.Enabled != nil && *prometheusValues.Enabled,
		Resources: vmov1.Resources{
			RequestMemory: "128Mi",
		},
		Storage: vmov1.Storage{},
	}
	setStorageSize(storage, &prometheus.Storage)
	if vmi != nil {
		prometheus.Storage = vmi.Spec.Prometheus.Storage
	}

	return prometheus
}

func setStorageSize(storage *common.ResourceRequestValues, storageObject *vmov1.Storage) {
	if storage == nil {
		storageObject.Size = "50Gi"
	} else {
		storageObject.Size = storage.Storage
	}
}
