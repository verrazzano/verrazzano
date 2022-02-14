// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerocidns

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager-ocidns"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "cert-manager"

// certManagerOciDnsComponent represents an CertManager component
type certManagerOciDNSComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerOciDNSComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerOciDNSComponent{
		helm.HelmComponent{
			Dependencies:            []string{certmanager.ComponentName},
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "cert-manager-webhook-oci"),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "cert-manager-ocidns-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_0_0,
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerOciDNSComponent) IsEnabled(ctx spi.ComponentContext) bool {
	return isCertManagerEnabled(ctx) && isOciDNSEnabled(ctx)
}

// IsReady component check
func (c certManagerOciDNSComponent) IsReady(ctx spi.ComponentContext) bool {
	if !isCertManagerEnabled(ctx) {
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return isCertManagerOciDNSReady(ctx)
	}
	return false
}

// IsCertManagerEnabled returns true if the cert-manager is enabled, which is the default
func isCertManagerEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// isOciDNSEnabled returns true if the oci-dns is enabled
func isOciDNSEnabled(compContext spi.ComponentContext) bool {
	dns := compContext.EffectiveCR().Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return true
	}
	return false
}
