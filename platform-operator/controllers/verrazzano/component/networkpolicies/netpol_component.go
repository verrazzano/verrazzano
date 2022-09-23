// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// The networkpolicies component is needed to apply network policies during install and upgrade before
// any other components are installed or upgraded. This removes any race conditions that used
// to occur when network polices where installed in parallel, after other components were
// already installed.  This component must be first in registry.go so that it runs first during
// upgrade.
//
// Most of the network policies were in verrazzano chart previous to the existence of this code, but not all of them.
// Some policies are specified in other charts, like verrazzano-monitoring-operator.  This is ideally what
// should happen, but for now, putting the network policies into a dedicated chart is a improvement.
//
// This code changes the ownership of the existing network policies from the verrazzano chart to the
// verrazzano-network-policies chart.  This is done by modifying the helm annotations in each NetworkPolicy
// resource thereby changing the Helm release ownership.
//
// There are additional network policies that are created dynamically.  For example, VOA creates a NetworkPolicy
// to an application in the mesh to access the Istio control plane (istiod).
//
// Note that there is no NetworkPolicy component in Verrazzano CR (API).  The user has no control over this
// component, it is always enabled. However, the network policies are only enforced if a CNI with a NetworkPolicy
// controller is running, such as Calico.

package networkpolicies

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"path/filepath"

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
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
			AppendOverridesFunc:       appendOverrides,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
		},
	}
}

// IsEnabled always returns true since network policies are always installed
func (c networkPoliciesComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return true
}

// PreInstall performs pre-install actions
func (c networkPoliciesComponent) PreInstall(ctx spi.ComponentContext) error {
	// Create all namespaces needed by network policies
	common.CreateAndLabelNamespaces(ctx)

	// Associate the network policies to the verrazzano-network-policies release
	err := associateNetworkPoliciesWithHelm(ctx)
	if err != nil {
		return err
	}

	return c.HelmComponent.PreInstall(ctx)
}

// PostInstall performs post-install actions
func (c networkPoliciesComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	return c.HelmComponent.PostInstall(ctx)
}

// PreUpgrade performs pre-upgrade actions
func (c networkPoliciesComponent) PreUpgrade(ctx spi.ComponentContext) error {
	// Create all namespaces needed by network policies
	common.CreateAndLabelNamespaces(ctx)

	// Associate the network policies to the verrazzano-network-policies release
	err := associateNetworkPoliciesWithHelm(ctx)
	if err != nil {
		return err
	}

	return c.HelmComponent.PreUpgrade(ctx)
}

// PostUpgrade performs post-upgrade actions
func (c networkPoliciesComponent) PostUpgrade(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	return c.HelmComponent.PostUpgrade(ctx)
}

// PreUninstall performs pre-uninstall actions
func (c networkPoliciesComponent) PreUninstall(ctx spi.ComponentContext) error {
	// Remove the helm "keep" policy so that the helm resources can get deleted
	err := removeResourcePolicyFromHelm(ctx)
	if err != nil {
		return err
	}

	return c.HelmComponent.PreUninstall(ctx)
}
