// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/controller"
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
	ComponentName = cmcommon.CertManagerOCIDNSComponentName

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
type certManagerOciDNSComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerOciDNSComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerOciDNSComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), componentChartName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
			GetInstallOverridesFunc:   GetOverrides,
			AppendOverridesFunc:       appendOCIDNSOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			Dependencies:              []string{networkpolicies.ComponentName, controller.ComponentName},
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

func (c certManagerOciDNSComponent) PreInstall(ctx spi.ComponentContext) error {
	effectiveCR := ctx.EffectiveCR()
	if err := common.CopyOCIDNSSecret(ctx, getClusterResourceNamespace(effectiveCR)); err != nil {
		return err
	}
	if err := common.CopyOCIDNSSecret(ctx, constants.CertManagerNamespace); err != nil {
		return err
	}
	return nil
}

func getClusterResourceNamespace(cr *vzapi.Verrazzano) string {
	if cr.Spec.Components.CertManager != nil && cr.Spec.Components.CertManager.Webhook != nil {
		return cr.Spec.Components.CertManager.Webhook.ClusterResourceNamespace
	}
	return constants.CertManagerNamespace
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerOciDNSComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsCertManagerOCIDNSWebhookEnabled(effectiveCR)
}

func (c certManagerOciDNSComponent) PostUninstall(ctx spi.ComponentContext) error {
	return c.postUninstall(ctx)
}

// IsReady component check
func (c certManagerOciDNSComponent) IsReady(ctx spi.ComponentContext) bool {
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
func (c certManagerOciDNSComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	certManager := ctx.EffectiveCR().Spec.Components.CertManager
	if certManager != nil && certManager.Webhook != nil {
		if certManager.Webhook.MonitorChanges != nil {
			return *certManager.Webhook.MonitorChanges
		}
		return true
	}
	return false
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		certManager := effectiveCR.Spec.Components.CertManager
		if certManager != nil && certManager.Webhook != nil {
			return certManager.Webhook.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	certManager := effectiveCR.Spec.Components.CertManager
	if certManager != nil && certManager.Webhook != nil {
		return certManager.Webhook.ValueOverrides
	}
	return []v1beta1.Overrides{}
}
