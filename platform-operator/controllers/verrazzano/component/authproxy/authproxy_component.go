// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"path/filepath"

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
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
		},
	}
}

// IsEnabled authProxyComponent-specific enabled check for installation
func (c authProxyComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.AuthProxy
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c authProxyComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// IsReady component check
func (c authProxyComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isAuthProxyReady(ctx)
	}
	return false
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (c authProxyComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	ingressNames := []types.NamespacedName{
		{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.VzConsoleIngress,
		},
	}
	return ingressNames
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

// PreUpgrade performs any required pre upgrade operations
func (c authProxyComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return authproxyPreHelmOps(ctx)
}
