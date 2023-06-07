// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "prometheus-node-exporter"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the JSON name of the component in the CRD
const ComponentJSONName = "prometheusNodeExporter"

const chartDir = "prometheus-community/prometheus-node-exporter"

var valuesFile = fmt.Sprintf("%s-values.yaml", ComponentName)

type prometheusNodeExporterComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return prometheusNodeExporterComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), chartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "serviceAccount.imagePullSecrets[0].name",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), valuesFile),
			Dependencies:              []string{promoperator.ComponentName, fluentoperator.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DaemonsetNames: []types.NamespacedName{
					{
						Name:      daemonsetName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsEnabled returns true if the Prometheus Node-Exporter is explicitly enabled in the Verrazzano CR, otherwise
// it returns true if the Prometheus component is enabled.
func (c prometheusNodeExporterComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsNodeExporterEnabled(effectiveCR)
}

// IsReady checks if the Prometheus Node-Exporter deployment is ready
func (c prometheusNodeExporterComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isPrometheusNodeExporterReady(ctx)
	}
	return false
}

// PreInstall updates resources necessary for the Prometheus Node-Exporter Component installation
func (c prometheusNodeExporterComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstall(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

// AppendOverrides appends install overrides for the Prometheus Node Exporter component's Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Only enable the node exporter's ServiceMonitor if Prometheus Operator is enabled in this install
	ctx.Log().Debug("Appending service monitor override for the Prometheus Node Exporter component")
	if vzcr.IsPrometheusOperatorEnabled(ctx.EffectiveCR()) {
		kvs = append(kvs, bom.KeyValue{
			Key: "prometheus.monitor.enabled", Value: "true",
		})
	}
	return kvs, nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c prometheusNodeExporterComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.PrometheusNodeExporter != nil {
		if ctx.EffectiveCR().Spec.Components.PrometheusNodeExporter.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.PrometheusNodeExporter.MonitorChanges
		}
	}
	return true
}

// PostInstall creates/updates associated resources after this component is installed
func (c prometheusNodeExporterComponent) PostInstall(ctx spi.ComponentContext) error {
	return createOrUpdateNetworkPolicies(ctx)
}

// PostUpgrade creates/updates associated resources after this component is installed
func (c prometheusNodeExporterComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return createOrUpdateNetworkPolicies(ctx)
}
