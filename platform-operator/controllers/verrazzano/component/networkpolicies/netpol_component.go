// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Note that there is no NetworkPolicy component in Verrazzano CR.
// This component is needed to apply network policies during install and upgrade.

package networkpolicies

import (
	"fmt"
	"path/filepath"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

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
			ReleaseName:    ComponentName,
			JSONName:       ComponentJSONName,
			ChartDir:       filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace: ComponentNamespace,
			//AppendOverridesFunc:       verrazzano.AppendVerrazzanoOverrides,
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

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c networkPoliciesComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c networkPoliciesComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// IsReady component check
func (c networkPoliciesComponent) IsReady(ctx spi.ComponentContext) bool {
	return c.HelmComponent.IsReady(ctx)
}

// getOverrides returns install overrides for a component
func getOverrides(object runtime.Object) interface{} {
	return []vzapi.Overrides{}
}
