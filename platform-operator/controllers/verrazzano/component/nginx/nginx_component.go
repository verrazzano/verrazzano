// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

// NginxComponent represents an Nginx component
type NginxComponent struct {
	helmComponent helm.HelmComponent
}

// Verify that NginxComponent implements Component
var _ spi.Component = NginxComponent{}

// NewComponent returns a new Nginx component
func NewComponent() spi.Component {
	return NginxComponent{
		helmComponent: helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "ingress-nginx"), // Note name is different than release name
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), ValuesFileOverride),
			PreInstallFunc:          PreInstall,
			AppendOverridesFunc:     AppendOverrides,
			PostInstallFunc:         PostInstall,
			Dependencies:            []string{istio.ComponentName},
			ReadyStatusFunc:         IsReady,
		},
	}
}

// --------------------------------------
// ComponentInfo interface functions
// --------------------------------------

// Log returns the logger for the context
func (c NginxComponent) Name() string {
	return c.helmComponent.Name()
}

// Log returns the logger for the context
func (c NginxComponent) GetDependencies() []string {
	return c.helmComponent.GetDependencies()
}

// IsReady Indicates whether or not a component is available and ready
func (c NginxComponent) IsReady(context spi.ComponentContext) bool {
	return c.helmComponent.IsReady(context)
}

// --------------------------------------
// ComponentInstaller interface functions
// --------------------------------------

// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
// - scaffolding while we move components from the scripts to the operator
func (c NginxComponent) IsOperatorInstallSupported() bool {
	return c.helmComponent.IsOperatorInstallSupported()
}

// IsInstalled Indicates whether or not the component is installed
func (c NginxComponent) IsInstalled(context spi.ComponentContext) (bool, error) {
	return c.helmComponent.IsInstalled(context)
}

// PreInstall allows components to perform any pre-processing required prior to initial install
func (c NginxComponent) PreInstall(context spi.ComponentContext) error {
	return c.helmComponent.PreInstall(context)
}

// Install performs the initial install of a component
func (c NginxComponent) Install(context spi.ComponentContext) error {
	return c.helmComponent.Install(context)
}

// PostInstall allows components to perform any post-processing required after initial install
func (c NginxComponent) PostInstall(context spi.ComponentContext) error {
	return c.helmComponent.PostInstall(context)
}

// --------------------------------------
// ComponentUpgrader interface functions
// --------------------------------------

// PreUpgrade allows components to perform any pre-processing required prior to upgrading
func (c NginxComponent) PreUpgrade(context spi.ComponentContext) error {
	return c.helmComponent.PreUpgrade(context)
}

// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
func (c NginxComponent) Upgrade(context spi.ComponentContext) error {
	return c.helmComponent.Upgrade(context)
}

// PostUpgrade allows components to perform any post-processing required after upgrading
func (c NginxComponent) PostUpgrade(context spi.ComponentContext) error {
	return c.helmComponent.PostUpgrade(context)
}

// GetSkipUpgrade returns the value of the SkipUpgrade field
// - Scaffolding for now during the Istio 1.10.2 upgrade process
func (c NginxComponent) GetSkipUpgrade() bool {
	return c.helmComponent.GetSkipUpgrade()
}
