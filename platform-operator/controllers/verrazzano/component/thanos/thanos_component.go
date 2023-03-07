// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "thanos"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the JSON name of the Thanos component in CRD
const ComponentJSONName = "thanos"

// Availability Object Names
const (
	queryDeployment    = "thanos-query"
	frontendDeployment = "thanos-query-frontend"
)

type thanosComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return thanosComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "image.pullSecrets[0]",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "thanos-values.yaml"),
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      frontendDeployment,
						Namespace: ComponentNamespace,
					},
					{
						Name:      queryDeployment,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsReady component check for Thanos
func (t thanosComponent) IsReady(context spi.ComponentContext) bool {
	return t.HelmComponent.IsReady(context) && t.isThanosReady(context)
}

// isThanosReady returns true if the availability objects have the minimum number of expected replicas
func (t thanosComponent) isThanosReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), t.AvailabilityObjects.DeploymentNames, 1, prefix)
}

// IsEnabled Thanos enabled check for installation
func (t thanosComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsThanosEnabled(effectiveCR)
}
