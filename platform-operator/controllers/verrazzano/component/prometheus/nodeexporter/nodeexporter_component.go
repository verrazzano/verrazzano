// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "node-exporter"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the json name of the component in the CRD
const ComponentJSONName = "prometheusNodeExporter"

const chartDir = "prometheus-community/prometheus-node-exporter"

type prometheusNodeExporterComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return prometheusNodeExporterComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), chartDir),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "image.pullSecrets[0]",
			Dependencies:            []string{},
		},
	}
}

// IsEnabled returns true if the Prometheus Node-Exporter is enabled or if the component is not specified
// in the Verrazzano CR.
func (c prometheusNodeExporterComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.PrometheusNodeExporter
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady checks if the Prometheus Node-Exporter deployment is ready
func (c prometheusNodeExporterComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isPrometheusNodeExporterReady(ctx)
	}
	return false
}

// PreInstall updates resources necessary for the Prometheus Node-Exporter Component installation
func (c prometheusNodeExporterComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}
