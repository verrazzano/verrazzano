// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookoci

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

const (
	// ComponentName is the name of the component
	ComponentName = cmcommon.CertManagerWebhookOCIComponentName

	// ComponentJSONName is the Webhook component JSON name in the Verrazzano CR
	ComponentJSONName = "certManagerOCIWebhook"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// componentChartName is the Webhook Chart name
	componentChartName = "verrazzano-cert-manager-ocidns-webhook"

	// webhookDeploymentName is the Webhook deployment object name
	webhookDeploymentName = "cert-manager-ocidns-provider"
)

// certManagerOciDnsComponent represents an CertManager component
type certManagerWebhookOCIComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerWebhookOCIComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerWebhookOCIComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), componentChartName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
			GetInstallOverridesFunc:   GetOverrides,
			AppendOverridesFunc:       appendOCIDNSOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			Dependencies:              []string{networkpolicies.ComponentName, cmcommon.CertManagerComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      webhookDeploymentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

func (c certManagerWebhookOCIComponent) PreInstall(ctx spi.ComponentContext) error {
	effectiveCR := ctx.EffectiveCR()
	if err := common.CopyOCIDNSSecret(ctx, getClusterResourceNamespace(effectiveCR)); err != nil {
		return err
	}
	return nil
}

// IsEnabled returns true if the component is explicitly enabled OR if OCI DNS/LetsEncrypt are configured
func (c certManagerWebhookOCIComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsCertManagerWebhookOCIRequired(effectiveCR)
}

func (c certManagerWebhookOCIComponent) PostUninstall(ctx spi.ComponentContext) error {
	return c.postUninstall(ctx)
}

// IsReady component check
func (c certManagerWebhookOCIComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config PostInstall dry run")
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return isCertManagerOciDNSReady(ctx)
	}
	return false
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c certManagerWebhookOCIComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.CertManagerWebhookOCI != nil {
		if ctx.EffectiveCR().Spec.Components.CertManagerWebhookOCI.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.CertManagerWebhookOCI.MonitorChanges
		}
		return true
	}
	return false
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.CertManagerWebhookOCI != nil {
			return effectiveCR.Spec.Components.CertManagerWebhookOCI.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.CertManagerWebhookOCI != nil {
		return effectiveCR.Spec.Components.CertManagerWebhookOCI.ValueOverrides
	}
	return []v1beta1.Overrides{}
}

// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
func (c certManagerWebhookOCIComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	return c.ValidateInstallV1Beta1(vzV1Beta1)
}

// ValidateInstallV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be installed
func (c certManagerWebhookOCIComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if err := c.validateConfiguration(vz); err != nil {
		return err
	}
	return c.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c certManagerWebhookOCIComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	oldBeta := &v1beta1.Verrazzano{}
	newBeta := &v1beta1.Verrazzano{}
	if err := old.ConvertTo(oldBeta); err != nil {
		return err
	}
	if err := new.ConvertTo(newBeta); err != nil {
		return err
	}

	return c.ValidateUpdateV1Beta1(oldBeta, newBeta)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c certManagerWebhookOCIComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if err := c.validateConfiguration(new); err != nil {
		return err
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

func (c certManagerWebhookOCIComponent) validateConfiguration(_ *v1beta1.Verrazzano) error {
	//acmeConfig, err := vzcr.IsLetsEncryptConfig(new)
	//if err != nil {
	//	return err
	//}

	// NOTE: this validation doesn't seem to work right now, will need to revisit

	//explicitlyDisabled := false
	//if new.Spec.Components.CertManagerWebhookOCI != nil {
	//	explicitlyDisabled = !*new.Spec.Components.CertManagerWebhookOCI.Enabled
	//}
	//// If OCI DNS/LetsEncrypt and the ClusterIssuer are enabled, the OCI DNS webhook is required
	//// Note: we implicitly deploy this under required circumstances, but we want to ensure that users know this is
	//// an illegal configuration
	//if explicitlyDisabled && vzcr.IsOCIDNSEnabled(new) && acmeConfig && vzcr.IsClusterIssuerEnabled(new) {
	//	return fmt.Errorf("the %s component is required when using OCI DNS with an LetsEncrypt certificate issuer configuration", ComponentJSONName)
	//}
	return nil
}
