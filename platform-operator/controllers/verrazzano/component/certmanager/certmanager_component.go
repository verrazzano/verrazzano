// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager"

// certManagerComponent represents an CertManager component
type certManagerComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "cert-manager"),
			ChartNamespace:          ComponentName,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "cert-manager-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_0_0,
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerComponent) IsEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
