// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
)

const (
	// ComponentName is the name of the component
	ComponentName = "ingress-controller"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = "ingress-nginx"

	// ComponentJSONName is the josn name of the verrazzano component in CRD
	ComponentJSONName = "ingress"

	// nginxExternalIPKey is the nginxInstallArgs key for externalIPs
	nginxExternalIPKey = "controller.service.externalIPs"
)

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
			JSONName:                ComponentJSONName,
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
		},
	}
}

// IsEnabled nginx-specific enabled check for installation
func (c nginxComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Ingress
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady component check
func (c nginxComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isNginxReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.validateForExternalIPSWithNodePort(&new.Spec)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return c.validateForExternalIPSWithNodePort(&vz.Spec)
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (c nginxComponent) validateForExternalIPSWithNodePort(vz *vzapi.VerrazzanoSpec) error {
	// good if ingress is not set
	if vz.Components.Ingress == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Components.Ingress.Type != vzapi.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Components.Ingress.Type == vzapi.NodePort {
		if vz.Components.Ingress.NGINXInstallArgs != nil {
			return vzconfig.CheckArgs(vz.Components.Ingress.NGINXInstallArgs, nginxExternalIPKey, c.Name())
		}
		// if ingress args are not set at all, then external ips will not be set
		return fmt.Errorf("'nginxInstallArgs' cannot be empty. ExternalIPs needs to be specified here as type is NodePort")
	}

	return nil
}
