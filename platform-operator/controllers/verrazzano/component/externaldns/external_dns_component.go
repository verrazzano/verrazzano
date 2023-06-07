// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "external-dns"

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "dns"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.ExternalDNSNamespace

// legacyNamespace is the namespace used for external-dns in older releases
const legacyNamespace = vzconst.CertManagerNamespace

type externalDNSComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = &externalDNSComponent{}

func NewComponent() spi.Component {
	return &externalDNSComponent{
		HelmComponent: helm.HelmComponent{
			JSONName:                  ComponentJSONName,
			ReleaseName:               ComponentName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    imagePullSecretHelmKey,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "external-dns-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_0_0,
			Dependencies:              []string{"verrazzano-network-policies", fluentoperator.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,

			// Resolve the namespace dynamically
			ResolveNamespaceFunc:    resolveExernalDNSNamespace,
			IgnoreNamespaceOverride: false,
		},
	}
}

func (c *externalDNSComponent) PreInstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("%s PreInstall dry run", ComponentName)
		return nil
	}
	if err := preInstall(compContext); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(compContext)
}

func (c *externalDNSComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("%s IsReady dry run", ComponentName)
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return c.isExternalDNSReady(ctx)
	}
	return false
}

func (c *externalDNSComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsExternalDNSEnabled(effectiveCR)
}

// PostUninstall Clean up external-dns resources not removed by Uninstall()
func (c *externalDNSComponent) PostUninstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("%s PostUninstall dry run", ComponentName)
		return nil
	}
	return postUninstall(ctx.Log(), ctx.Client())
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c *externalDNSComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling an existing OCI DNS configuration is not allowed")
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c *externalDNSComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling an existing OCI DNS configuration is not allowed")
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c *externalDNSComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.DNS != nil {
		if ctx.EffectiveCR().Spec.Components.DNS.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.DNS.MonitorChanges
		}
		return true
	}
	return false
}

// IsAvailable indicates whether a component is Available for end users.
func (c *externalDNSComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available vzapi.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: c.getDeploymentNames(ctx)}).IsAvailable(ctx.Log(), ctx.Client())
}
