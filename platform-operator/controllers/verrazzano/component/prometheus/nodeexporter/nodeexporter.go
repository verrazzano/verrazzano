// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const daemonsetName = "prometheus-node-exporter" // Should match fullName override in prometheus-node-exporter-values.yaml

// isPrometheusNodeExporterReady checks if the Prometheus Node-Exporter daemonset is ready
func isPrometheusNodeExporterReady(ctx spi.ComponentContext) bool {
	sets := []types.NamespacedName{
		{
			Name:      daemonsetName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DaemonSetsAreReady(ctx.Log(), ctx.Client(), sets, 1, prefix)
}

// PreInstall implementation for the Prometheus Node-Exporter Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Node-Exporter preInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Node-Exporter", ComponentNamespace)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), prometheus.GetVerrazzanoMonitoringNamespace(), func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// GetOverrides returns install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.PrometheusNodeExporter != nil {
		return effectiveCR.Spec.Components.PrometheusNodeExporter.ValueOverrides
	}
	return []vzapi.Overrides{}
}
