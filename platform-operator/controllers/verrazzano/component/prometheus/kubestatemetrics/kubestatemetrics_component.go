// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "kube-state-metrics"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the json name of the component in the CRD
const ComponentJSONName = "kubeStateMetrics"

const chartDir = "prometheus-community/kube-state-metrics"

type kubeStateMetricsComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return kubeStateMetricsComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), chartDir),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "kube-state-metrics-values.yaml"),
			Dependencies:            []string{},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// IsEnabled returns true if kube-state-metrics is enabled or if the component is not specified
// in the Verrazzano CR.
func (c kubeStateMetricsComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.KubeStateMetrics
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsReady checks if the kube-state-metrics deployment is ready
func (c kubeStateMetricsComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isDeploymentReady(ctx)
	}
	return false
}

// PreInstall updates resources necessary for kube-state-metrics Component installation
func (c kubeStateMetricsComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c kubeStateMetricsComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.KubeStateMetrics != nil {
		if ctx.EffectiveCR().Spec.Components.KubeStateMetrics.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.KubeStateMetrics.MonitorChanges
		}
		return true
	}
	return false
}
