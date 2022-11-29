// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-cluster-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "clusterOperator"

type clusterOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return clusterOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0]",
			Dependencies:              []string{networkpolicies.ComponentName, rancher.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// PreInstall processing for the cluster-operator
func (c clusterOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return applyCRDs(ctx)
}

// PostInstall processing for the cluster-operator
func (c clusterOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	return c.postInstallUpgrade(ctx)
}

// PreUpgrade processing for the cluster-operator
func (c clusterOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return applyCRDs(ctx)
}

// PostUpgrade processing for the cluster-operator
func (c clusterOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return c.postInstallUpgrade(ctx)
}

func applyCRDs(ctx spi.ComponentContext) error {
	return common.ApplyCRDYaml(ctx, config.GetHelmClusterOpChartsDir())
}

// IsReady component check
func (c clusterOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return c.isClusterOperatorReady(context)
	}
	return false
}

// IsEnabled cluster operator enabled check for installation
func (c clusterOperatorComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsClusterOperatorEnabled(effectiveCR)
}
