// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager"

var hc = helm.HelmComponent{
	ReleaseName:             "cert-manager",
	ChartDir:                filepath.Join(config.GetThirdPartyDir(), "cert-manager"),
	ChartNamespace:          "cert-manager",
	IgnoreNamespaceOverride: true,
	ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "cert-manager-values.yaml"),
}

// CertManagerComponent represents an CertManager component
type CertManagerComponent struct {
}

// Verify that CertManagerComponent implements Component
var _ spi.Component = CertManagerComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return CertManagerComponent{}
}

// --------------------------------------
// ComponentInfo interface functions
// --------------------------------------

// Log returns the logger for the context
func (c CertManagerComponent) Name() string {
	return hc.Name()
}

// Log returns the logger for the context
func (c CertManagerComponent) GetDependencies() []string {
	return hc.GetDependencies()
}

// IsReady Indicates whether or not a component is available and ready
func (c CertManagerComponent) IsReady(context spi.ComponentContext) bool {
	return hc.IsReady(context)
}

// --------------------------------------
// ComponentInstaller interface functions
// --------------------------------------

// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
// - scaffolding while we move components from the scripts to the operator
func (c CertManagerComponent) IsOperatorInstallSupported() bool {
	return hc.IsOperatorInstallSupported()
}

// IsInstalled Indicates whether or not the component is installed
func (c CertManagerComponent) IsInstalled(context spi.ComponentContext) (bool, error) {
	return hc.IsInstalled(context)
}

// PreInstall allows components to perform any pre-processing required prior to initial install
func (c CertManagerComponent) PreInstall(context spi.ComponentContext) error {
	return hc.PreInstall(context)
}

// Install performs the initial install of a component
func (c CertManagerComponent) Install(context spi.ComponentContext) error {
	return hc.Install(context)
}

// PostInstall allows components to perform any post-processing required after initial install
func (c CertManagerComponent) PostInstall(context spi.ComponentContext) error {
	return hc.PostInstall(context)
}

// --------------------------------------
// ComponentUpgrader interface functions
// --------------------------------------

// PreUpgrade allows components to perform any pre-processing required prior to upgrading
func (c CertManagerComponent) PreUpgrade(context spi.ComponentContext) error {
	return hc.PreUpgrade(context)
}

// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
func (c CertManagerComponent) Upgrade(context spi.ComponentContext) error {
	return hc.Upgrade(context)
}

// PostUpgrade allows components to perform any post-processing required after upgrading
func (c CertManagerComponent) PostUpgrade(context spi.ComponentContext) error {
	return hc.PostUpgrade(context)
}

// GetSkipUpgrade returns the value of the SkipUpgrade field
// - Scaffolding for now during the Istio 1.10.2 upgrade process
func (c CertManagerComponent) GetSkipUpgrade() bool {
	return hc.GetSkipUpgrade()
}
