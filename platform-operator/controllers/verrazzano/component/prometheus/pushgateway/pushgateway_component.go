// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "prometheus-pushgateway"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the json name of the component in the CRD
const ComponentJSONName = "prometheusPushgateway"

const chartName = "prometheus-community/prometheus-pushgateway"

type prometheusPushgatewayComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return prometheusPushgatewayComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), chartName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "prometheus-pushgateway-values.yaml"),
			Dependencies:            []string{},
			AppendOverridesFunc:     AppendOverrides,
			GetHelmValueOverrides:   GetHelmOverrides,
		},
	}
}

// IsEnabled returns true if the Prometheus PrometheusPushgateway is enabled or if the component is not specified
// in the Verrazzano CR.
func (c prometheusPushgatewayComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.PrometheusPushgateway
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsReady checks if the Prometheus PrometheusPushgateway deployment is ready
func (c prometheusPushgatewayComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isPushgatewayReady(ctx)
	}
	return false
}

// PreInstall updates resources necessary for the Prometheus PrometheusPushgateway Component installation
func (c prometheusPushgatewayComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

// ValidateInstall verifies the installation of the Verrazzano object
func (c prometheusPushgatewayComponent) ValidateInstall(effectiveCR *vzapi.Verrazzano) error {
	if effectiveCR.Spec.Components.PrometheusPushgateway != nil {
		return vzapi.ValidateHelmValueOverrides(effectiveCR.Spec.Components.PrometheusPushgateway.ValueOverrides)
	}
	return nil
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (c prometheusPushgatewayComponent) ValidateUpgrade(effectiveCR *vzapi.Verrazzano) error {
	if effectiveCR.Spec.Components.PrometheusPushgateway != nil {
		return vzapi.ValidateHelmValueOverrides(effectiveCR.Spec.Components.PrometheusPushgateway.ValueOverrides)
	}
	return nil
}
