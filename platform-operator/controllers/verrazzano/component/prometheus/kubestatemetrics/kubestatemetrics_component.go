// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
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
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), chartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "imagePullSecrets[0].name",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "kube-state-metrics-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			Dependencies:              []string{promoperator.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled returns true if kube-state-metrics is enabled or if the component is not specified
// in the Verrazzano CR.
func (c kubeStateMetricsComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsKubeStateMetricsEnabled(effectiveCR)
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

// AppendOverrides appends install overrides for the Prometheus kube-state-metrics component's Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Only enable the ServiceMonitor if Prometheus Operator is enabled in this install
	ctx.Log().Debug("Appending service monitor override for the Prometheus kube-state-metrics component")
	if vzconfig.IsPrometheusOperatorEnabled(ctx.EffectiveCR()) {
		kvs = append(kvs, bom.KeyValue{
			Key: "prometheus.monitor.enabled", Value: "true",
		})
	}
	return kvs, nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c kubeStateMetricsComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	// we do not allow disabling this component once it has been enabled
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated (VZ v1beta1)
func (c kubeStateMetricsComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// we do not allow disabling this component once it has been enabled
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}
