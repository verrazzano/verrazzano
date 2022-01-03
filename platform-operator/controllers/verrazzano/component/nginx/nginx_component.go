// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

// nginxComponent represents an Nginx component
type nginxComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = nginxComponent{}

// NewComponent returns a new Nginx component
func NewComponent() spi.Component {
	return nginxComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "ingress-nginx"), // Note name is different than release name
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), ValuesFileOverride),
			PreInstallFunc:          PreInstall,
			AppendOverridesFunc:     AppendOverrides,
			PostInstallFunc:         PostInstall,
			Dependencies:            []string{istio.ComponentName},
			ReadyStatusFunc:         IsReady,
		},
	}
}

// IsEnabled nginx-specific enabled check for installation
func (c nginxComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Ingress
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
