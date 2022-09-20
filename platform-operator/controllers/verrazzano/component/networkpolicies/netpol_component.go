// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Note that there is no NetworkPolicy component in Verrazzano CR.
// This component is needed to apply network policies during install and upgrade.

package networkpolicies

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-network-policies"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentName is the JSON name of the component
const ComponentJSONName = "verrazzanoNetworkPolicies"

type networkPoliciesComponent struct {
	helm.HelmComponent
}

// NewComponent returns a new networkPoliciesComponent
// The network policies helm chart can use the same overrides as verrazznoa
func NewComponent() spi.Component {
	return networkPoliciesComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			AppendOverridesFunc:       appendOverrides,
			GetInstallOverridesFunc:   getOverrides,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
		},
	}
}

// IsEnabled WebLogic-specific enabled check for installation
func (c networkPoliciesComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsWebLogicOperatorEnabled(effectiveCR)
}

// IsInstalled component check - network policies are always applied
func (c networkPoliciesComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return true, nil
}

// IsReady component check
func (c networkPoliciesComponent) IsReady(ctx spi.ComponentContext) bool {
	return c.HelmComponent.IsReady(ctx)
}

// PreUpgrade dis-associates existing network policies from Verrazzano chart
func (c networkPoliciesComponent) PreUpgrade(ctx spi.ComponentContext) error {

	// TODO write this code, see authProxy

	return nil
}
