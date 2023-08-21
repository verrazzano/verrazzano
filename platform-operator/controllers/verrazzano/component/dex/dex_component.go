// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "dex"

// ComponentJSONName is the JSON name of the Dex component in CRD
const ComponentJSONName = "dex"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.DexNamespace

// DexComponent represents an Dex component
type DexComponent struct {
	helm.HelmComponent
}

// Verify that DexComponent implements Component
var _ spi.Component = DexComponent{}

// NewComponent returns a new Keycloak component
func NewComponent() spi.Component {
	return DexComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "dex-values.yaml"),
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName},
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendDexOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
			// IngressNames
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// IsReady component check for Thanos
func (t DexComponent) IsReady(ctx spi.ComponentContext) bool {
	return t.HelmComponent.IsReady(ctx) && t.isDexReady(ctx)
}

// isThanosReady returns true if the availability objects that exist, have the minimum number of expected replicas
func (t DexComponent) isDexReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	// If a Dex component is enabled, ensure the deployment exists after Helm installation
	// TODO: Check if replicas are ready, find out whether there is anything we need to care
	deploymentsToCheck := t.getEnabledDeployments(ctx)
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deploymentsToCheck, 1, prefix)
}

func (t DexComponent) getEnabledDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	enabledDeployments := []types.NamespacedName{}
	for _, deploymentName := range t.AvailabilityObjects.DeploymentNames {
		if exists, err := ready.DoesDeploymentExist(ctx.Client(), deploymentName); err == nil && exists {
			enabledDeployments = append(enabledDeployments, deploymentName)
		}
	}
	return enabledDeployments
}

// IsEnabled Dex enabled check for installation
func (t DexComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsDexEnabled(effectiveCR)
}

// PreInstall handles the pre-install operations for the Dex component
func (t DexComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return t.HelmComponent.PreInstall(ctx)
}

// PreUpgrade handles the pre-upgrade operations for the Dex component
func (t DexComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return t.HelmComponent.PreUpgrade(ctx)
}
