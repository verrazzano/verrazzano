// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"

	corev1 "k8s.io/api/core/v1"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
)

// ComponentName is the name of the component
// NOTE, ComponentNamespace namespace is determined at runtime, see nginxutil.go
const ComponentName = "ingress-controller"

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "ingressNGINX"

// nginxExternalIPKey is the nginxInstallArgs key for externalIPs
const nginxExternalIPKey = "controller.service.externalIPs"

const nginxExternalIPJsonPath = "controller.service.externalIPs.0"

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
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), "ingress-nginx"), // Note name is different than release name
			ChartNamespace:            nginxutil.IngressNGINXNamespace(),
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), ValuesFileOverride),
			PreInstallFunc:            PreInstall,
			AppendOverridesFunc:       AppendOverrides,
			PostInstallFunc:           PostInstall,
			Dependencies:              []string{networkpolicies.ComponentName, istio.ComponentName, fluentoperator.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ControllerName,
						Namespace: nginxutil.IngressNGINXNamespace(),
					},
					{
						Name:      backendName,
						Namespace: nginxutil.IngressNGINXNamespace(),
					},
				},
			},
		},
	}
}

// IsEnabled nginx-specific enabled check for installation
func (c nginxComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsNGINXEnabled(effectiveCR)
}

// IsReady component check
func (c nginxComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isNginxReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if err := c.HelmComponent.ValidateUpdate(old, new); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePort(new)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	if err := c.HelmComponent.ValidateInstall(vz); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePort(vz)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if err := c.HelmComponent.ValidateInstallV1Beta1(vz); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePortV1Beta1(vz)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	if err := c.HelmComponent.ValidateUpdateV1Beta1(old, new); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePortV1Beta1(new)
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (c nginxComponent) validateForExternalIPSWithNodePort(vz *vzapi.Verrazzano) error {
	// good if ingress is not set
	if vz.Spec.Components.Ingress == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Spec.Components.Ingress.Type != vzapi.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Spec.Components.Ingress.Type == vzapi.NodePort {
		return vzconfig.CheckExternalIPsArgs(vz.Spec.Components.Ingress.NGINXInstallArgs, vz.Spec.Components.Ingress.ValueOverrides, nginxExternalIPKey, nginxExternalIPJsonPath, c.Name(), vz.Namespace)
	}

	return nil
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (c nginxComponent) validateForExternalIPSWithNodePortV1Beta1(vz *v1beta1.Verrazzano) error {
	// good if ingress is not set
	if vz.Spec.Components.IngressNGINX == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Spec.Components.IngressNGINX.Type != v1beta1.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Spec.Components.IngressNGINX.Type == v1beta1.NodePort {
		return vzconfig.CheckExternalIPsOverridesArgs(vz.Spec.Components.IngressNGINX.ValueOverrides, nginxExternalIPJsonPath, c.Name(), vz.Namespace)
	}

	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c nginxComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Ingress != nil {
		if ctx.EffectiveCR().Spec.Components.Ingress.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Ingress.MonitorChanges
		}
		return true
	}
	return false
}

// PostUninstall processing for NGINX
func (c nginxComponent) PostUninstall(context spi.ComponentContext) error {
	res := resource.Resource{
		Name:   nginxutil.IngressNGINXNamespace(),
		Client: context.Client(),
		Object: &corev1.Namespace{},
		Log:    context.Log(),
	}
	// Remove finalizers from the ingress-nginx namespace to avoid hanging namespace deletion
	// and delete the namespace
	return res.RemoveFinalizersAndDelete()
}
