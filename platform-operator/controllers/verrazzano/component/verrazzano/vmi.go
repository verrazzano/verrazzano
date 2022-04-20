// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vmi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	nodeExporter = "node-exporter"
	system       = "system"
)

//createVMI instantiates the VMI resource and the Grafana Dashboards configmap
func createVMI(ctx spi.ComponentContext) error {
	if !vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		return nil
	}

	effectiveCR := ctx.EffectiveCR()
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}

	if err := createGrafanaConfigMaps(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("failed to create grafana configmaps: %v", err)
	}
	values := &verrazzanoValues{}
	if err := appendVerrazzanoValues(ctx, values); err != nil {
		return ctx.Log().ErrorfNewErr("failed to get Verrazzano values: %v", err)
	}
	storage, err := vmi.FindStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to get storage overrides: %v", err)
	}
	vmi := vmi.NewVMI()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), vmi, func() error {
		var existingVMI *vmov1.VerrazzanoMonitoringInstance = nil
		if len(vmi.Spec.URI) > 0 {
			existingVMI = vmi.DeepCopy()
		}

		vmi.Labels = map[string]string{
			"k8s-app":            "verrazzano.io",
			"verrazzano.binding": system,
		}
		cr := ctx.EffectiveCR()
		vmi.Spec.URI = fmt.Sprintf("vmi.system.%s.%s", values.Config.EnvName, dnsSuffix)
		vmi.Spec.IngressTargetDNSName = fmt.Sprintf("verrazzano-ingress.%s.%s", values.Config.EnvName, dnsSuffix)
		vmi.Spec.ServiceType = "ClusterIP"
		vmi.Spec.AutoSecret = true
		vmi.Spec.SecretsName = ComponentName
		vmi.Spec.CascadingDelete = true
		vmi.Spec.Grafana = newGrafana(cr, storage, existingVMI)
		vmi.Spec.Prometheus = newPrometheus(cr, storage, existingVMI)
		return nil
	})
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to update VMI: %v", err)
	}
	return nil
}

func newGrafana(cr *vzapi.Verrazzano, storage *vmi.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Grafana {
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

func newPrometheus(cr *vzapi.Verrazzano, storage *vmi.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance) vmov1.Prometheus {
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

func setStorageSize(storage *vmi.ResourceRequestValues, storageObject *vmov1.Storage) {
	if storage == nil {
		storageObject.Size = "50Gi"
	} else {
		storageObject.Size = storage.Storage
	}
}
