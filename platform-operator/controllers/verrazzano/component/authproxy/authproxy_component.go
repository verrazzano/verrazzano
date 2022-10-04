// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"path/filepath"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-authproxy"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "authProxy"

type authProxyComponent struct {
	helm.HelmComponent
}

// Verify that AuthProxyComponent implements Component
var _ spi.Component = authProxyComponent{}

// NewComponent returns a new authProxyComponent component
func NewComponent() spi.Component {
	return authProxyComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0]",
			GetInstallOverridesFunc:   GetOverrides,
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName},
			Certificates: []types.NamespacedName{
				{Name: constants.VerrazzanoIngressSecret, Namespace: ComponentNamespace},
			},
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.VzConsoleIngress,
				},
			},
		},
	}
}

// IsEnabled authProxyComponent-specific enabled check for installation
func (c authProxyComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsAuthProxyEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c authProxyComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c authProxyComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// IsReady component check
func (c authProxyComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isAuthProxyReady(ctx)
	}
	return false
}

// PreInstall - actions to perform prior to installing this component
func (c authProxyComponent) PreInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debug("AuthProxy pre-install")

	err := authproxyPreHelmOps(ctx)
	if err != nil {
		return err
	}

	// Temporary work around for installer bug of calling pre-install after a component is installed
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		ctx.Log().Oncef("Component %s already installed, skipping PreInstall checks", ComponentName)
		return nil
	}

	return nil
}

// Uninstall Authproxy to handle upgrade case where Authproxy was not its own helm chart.
// In that case, we need to delete the Authproxy resources explicitly
func (c authProxyComponent) Uninstall(context spi.ComponentContext) error {
	installed, err := c.HelmComponent.IsInstalled(context)
	if err != nil {
		return err
	}

	// If the helm chart is installed, then uninstall
	if installed {
		return c.HelmComponent.Uninstall(context)
	}

	// Attempt to delete the Authproxy resources
	rs := getAuthproxyManagedResources()
	for _, r := range rs {
		err := resource.Resource{
			Name:      r.NamespacedName.Name,
			Namespace: r.NamespacedName.Namespace,
			Client:    context.Client(),
			Object:    r.Obj,
			Log:       context.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}

	return nil
}

// PreUpgrade performs any required pre upgrade operations
func (c authProxyComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return authproxyPreHelmOps(ctx)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c authProxyComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.AuthProxy != nil {
		if ctx.EffectiveCR().Spec.Components.AuthProxy.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.AuthProxy.MonitorChanges
		}
		return true
	}
	return false
}
