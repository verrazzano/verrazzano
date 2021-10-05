// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"path/filepath"
)

const (
	rootDir                = "/verrazzano"
	platformDirSuffix      = "/platform-operator"
	installDirSuffix       = "/platform-operator/scripts/install"
	thirdPartyDirSuffix    = "/platform-operator/thirdparty/charts"
	helmConfigDirSuffix    = "/platform-operator/helm_config"
	helmChartsDirSuffix    = "/platform-operator/helm_config/charts"
	helmVzChartsDirSuffix  = "/platform-operator/helm_config/charts/verrazzano"
	helmOverridesDirSuffix = "/platform-operator/helm_config/overrides"
)

const defaultBomFilename = "verrazzano-bom.json"

// Global override for the default BOM file path
var bomFilePathOverride string

// TestHelmConfigDir is needed for unit tests
var TestHelmConfigDir string

// OperatorConfig specfies the Verrazzano Platform Operator Config
type OperatorConfig struct {

	// The CertDir directory containing tls.crt and tls.key
	CertDir string

	// InitWebhooks enables initialzation of webhooks for the operator
	InitWebhooks bool

	// MetricsAddr is the address the metric endpoint binds to
	MetricsAddr string

	// LeaderElectionEnabled  enables/disables ensuring that there is only one active controller manager
	LeaderElectionEnabled bool

	// VersionCheckEnabled enables/disables version checking for upgrade.
	VersionCheckEnabled bool

	// WebhooksEnabled enables/disables Webhooks for the operator
	WebhooksEnabled bool

	// WebhookValidationEnabled enables/disables webhook validation without removing the webhook itself
	WebhookValidationEnabled bool

	// VerrazzanoRootDir is the root verrazzano directory in the image
	VerrazzanoRootDir string

	// DryRun Run installs in a dry-run mode
	DryRun bool
}

// The singleton instance of the operator config
var instance = OperatorConfig{
	CertDir:                  "/etc/webhook/certs",
	InitWebhooks:             false,
	MetricsAddr:              ":8080",
	LeaderElectionEnabled:    false,
	VersionCheckEnabled:      true,
	WebhooksEnabled:          true,
	WebhookValidationEnabled: true,
	VerrazzanoRootDir:        rootDir,
}

// Set saves the operator config.  This should only be called at operator startup and during unit tests
func Set(config OperatorConfig) {
	instance = config
}

// Get returns the singleton instance of the operator config
func Get() OperatorConfig {
	return instance
}

// GetHelmConfigDir returns the helm config dir
func GetHelmConfigDir() string {
	if TestHelmConfigDir != "" {
		return TestHelmConfigDir
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmConfigDirSuffix)
}

// GetHelmChartsDir returns the helm charts dir
func GetHelmChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmChartsDirSuffix)
}

// GetHelmVzChartsDir returns the Verrazzano helm charts dir
func GetHelmVzChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/verrazzano")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmVzChartsDirSuffix)
}

// GetHelmOverridesDir returns the helm overrides dir
func GetHelmOverridesDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/overrides")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmOverridesDirSuffix)
}

// GetInstallDir returns the install dir
func GetInstallDir() string {
	return filepath.Join(instance.VerrazzanoRootDir, installDirSuffix)
}

// GetPlatformDir returns the platform dir
func GetPlatformDir() string {
	return filepath.Join(instance.VerrazzanoRootDir, platformDirSuffix)
}

// GetThirdPartyDir returns the thirdparty dir
func GetThirdPartyDir() string {
	return filepath.Join(instance.VerrazzanoRootDir, thirdPartyDirSuffix)
}

// SetDefaultBomFilePath Sets the global default location for the BOM file
func SetDefaultBomFilePath(p string) {
	bomFilePathOverride = p
}

// GetDefaultBOMFilePath returns BOM file path for the platform operator
func GetDefaultBOMFilePath() string {
	if bomFilePathOverride != "" {
		return bomFilePathOverride
	}
	return filepath.Join(GetPlatformDir(), defaultBomFilename)
}

func GetInjectedSystemNamespaces() []string {
	return []string{constants.VerrazzanoSystemNamespace, constants.IngressNginxNamespace, constants.KeycloakNamespace}
}
