// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "prometheus-pushgateway"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the JSON name of the component in the CRD
const ComponentJSONName = "prometheusPushgateway"

const chartName = "prometheus-community/prometheus-pushgateway"

type prometheusPushgatewayComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return prometheusPushgatewayComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), chartName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "prometheus-pushgateway-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			Dependencies:              []string{promoperator.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      deploymentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsEnabled returns true if the Prometheus PrometheusPushgateway is enabled or if the component is not specified
// in the Verrazzano CR.
func (c prometheusPushgatewayComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsPrometheusPushgatewayEnabled(effectiveCR)
}

// IsReady checks if the Prometheus PrometheusPushgateway deployment is ready
func (c prometheusPushgatewayComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isPushgatewayReady(ctx)
	}
	return false
}

// PreInstall updates resources necessary for the Prometheus PrometheusPushgateway Component installation
func (c prometheusPushgatewayComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstall(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c prometheusPushgatewayComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.PrometheusPushgateway != nil {
		if ctx.EffectiveCR().Spec.Components.PrometheusPushgateway.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.PrometheusPushgateway.MonitorChanges
		}
		return true
	}
	return false
}

// AppendOverrides appends install overrides for the Prometheus PrometheusPushgateway component's Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Only enable the ServiceMonitor if Prometheus Operator is enabled in this install
	ctx.Log().Debug("Appending service monitor override for the Prometheus PrometheusPushgateway component")
	if vzcr.IsPrometheusOperatorEnabled(ctx.EffectiveCR()) {
		kvs = append(kvs, bom.KeyValue{
			Key: "serviceMonitor.enabled", Value: "true",
		})
	}
	return kvs, nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c prometheusPushgatewayComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	// we do not allow disabling this component once it has been enabled
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated (VZ v1beta1)
func (c prometheusPushgatewayComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// we do not allow disabling this component once it has been enabled
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}
