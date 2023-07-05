// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-cluster-agent"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "clusterAgent"

type clusterAgentComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return clusterAgentComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendClusterAgentOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0]",
			Dependencies:              []string{networkpolicies.ComponentName, oam.ComponentName, istio.ComponentName},
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

// IsEnabled checks to see if the cluster agent component is enabled
func (c clusterAgentComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsClusterAgentEnabled(effectiveCR)
}

// isClusterAgentReady checks if the cluster agent deployment is ready
func (c clusterAgentComponent) isClusterAgentReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
}

// IsReady checks if the cluster agent deployment is ready
func (c clusterAgentComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isClusterAgentReady(ctx)
	}
	return false
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c clusterAgentComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.ClusterAgent != nil {
		if ctx.EffectiveCR().Spec.Components.ClusterAgent.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.ClusterAgent.MonitorChanges
		}
		return true
	}
	return false
}
