// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"fmt"
	k8s "github.com/verrazzano/verrazzano/platform-operator/internal/nodeport"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"path/filepath"
	"reflect"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "ingress-nginx"

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "ingress"

// nginxComponent represents an Nginx component
type nginxComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = &nginxComponent{}

// NewComponent returns a new Nginx component
func NewComponent() spi.Component {
	return &nginxComponent{
		helm.HelmComponent{
			ComponentInfoImpl: spi.ComponentInfoImpl{
				ComponentName:           ComponentName,
				SupportsOperatorInstall: true,
				Dependencies:            []string{istio.ComponentName},
			},
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "ingress-nginx"), // Note name is different than release name
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
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
func (c *nginxComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Ingress
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady component check
func (c *nginxComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isNginxReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Block all changes for now, particularly around storage changes
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if !reflect.DeepEqual(c.getInstallArgs(old), c.getInstallArgs(new)) {
		return fmt.Errorf("Updates to nginxInstallArgs not allowed for %s", ComponentJSONName)
	}
	if !reflect.DeepEqual(c.getPorts(old), c.getPorts(new)) {
		return fmt.Errorf("Updates to ports not allowed for %s", ComponentJSONName)
	}
	oldType, err := vzconfig.GetServiceType(old)
	if err != nil {
		return err
	}
	newType, err := vzconfig.GetServiceType(new)
	if err != nil {
		return err
	}
	if oldType != newType {
		return fmt.Errorf("Updates to service type not allowed for %s", ComponentJSONName)
	}
	return nil
}

func (c nginxComponent) getInstallArgs(vz *vzapi.Verrazzano) []vzapi.InstallArgs {
	if vz != nil && vz.Spec.Components.Ingress != nil {
		return vz.Spec.Components.Ingress.NGINXInstallArgs
	}
	return nil
}

func (c nginxComponent) getPorts(vz *vzapi.Verrazzano) []corev1.ServicePort {
	if vz != nil && vz.Spec.Components.Ingress != nil {
		return vz.Spec.Components.Ingress.Ports
	}
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return k8s.ValidateForExternalIPSWithNodePort(&vz.Spec, c.Name())
}
