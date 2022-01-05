// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package externaldns

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

type externalDNSComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = externalDNSComponent{}

func NewComponent() spi.Component {
	return externalDNSComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          "cert-manager",
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  imagePullSecretHelmKey,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "external-dns-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_0_0,
		},
	}
}

func (e externalDNSComponent) PreInstall(compContext spi.ComponentContext) error {
	return preInstall(compContext)
}

func (e externalDNSComponent) IsReady(compContext spi.ComponentContext) bool {
	return isReady(compContext)
}

func (e externalDNSComponent) IsEnabled(compContext spi.ComponentContext) bool {
	dns := compContext.EffectiveCR().Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return true
	}
	return false
}
