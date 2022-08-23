// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"fmt"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "external-dns"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "cert-manager"

type externalDNSComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = externalDNSComponent{}

func NewComponent() spi.Component {
	return externalDNSComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    imagePullSecretHelmKey,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "external-dns-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_0_0,
			Dependencies:              []string{},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

func (e externalDNSComponent) PreInstall(compContext spi.ComponentContext) error {
	return preInstall(compContext)
}

func (e externalDNSComponent) IsReady(ctx spi.ComponentContext) bool {
	if e.HelmComponent.IsReady(ctx) {
		return isExternalDNSReady(ctx)
	}
	return false
}

func (e externalDNSComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	dns := effectiveCR.Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return true
	}
	return false
}

// PostUninstall Clean up external-dns resources not removed by Uninstall()
func (e externalDNSComponent) PostUninstall(ctx spi.ComponentContext) error {
	return postUninstall(ctx.Log(), ctx.Client())
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (e externalDNSComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if e.IsEnabled(old) && !e.IsEnabled(new) {
		return fmt.Errorf("Disabling an existing OCI DNS configuration is not allowed")
	}
	return e.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (e externalDNSComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (e externalDNSComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.DNS != nil {
		if ctx.EffectiveCR().Spec.Components.DNS.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.DNS.MonitorChanges
		}
		return true
	}
	return false
}
