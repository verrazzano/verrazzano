// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/nginxutil"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

const (
	rootDir                      = "/verrazzano"
	platformDirSuffix            = "/platform-operator"
	profilesDirSuffix            = "/platform-operator/manifests/profiles"
	installDirSuffix             = "/platform-operator/scripts/install"
	thirdPartyDirSuffix          = "/platform-operator/thirdparty/charts"
	thirdPartyManifestsDirSuffix = "/platform-operator/thirdparty/manifests"
	helmConfigDirSuffix          = "/platform-operator/helm_config"
	helmChartsDirSuffix          = "/platform-operator/helm_config/charts"
	helmVPOChartsDirSuffix       = "/platform-operator/helm_config/charts/verrazzano-platform-operator"
	helmVMOChartsDirSuffix       = "/platform-operator/helm_config/charts/verrazzano-monitoring-operator"
	helmAppOpChartsDirSuffix     = "/platform-operator/helm_config/charts/verrazzano-application-operator"
	helmClusterOpChartsDirSuffix = "/platform-operator/helm_config/charts/verrazzano-cluster-operator"
	helmKialiChartsDirSuffix     = "/platform-operator/thirdparty/charts/kiali-server"
	helmPromOpChartsDirSuffix    = "/platform-operator/thirdparty/charts/prometheus-community/kube-prometheus-stack"
	helmOamChartsDirSuffix       = "/platform-operator/thirdparty/charts/oam-kubernetes-runtime"
	helmOverridesDirSuffix       = "/platform-operator/helm_config/overrides"
)

const defaultBomFilename = "verrazzano-bom.json"

// Global override for the default BOM file path
var bomFilePathOverride string

// TestHelmConfigDir is needed for unit tests
var TestHelmConfigDir string

// TestProfilesDir is needed for unit tests
var TestProfilesDir string

// TestThirdPartyManifestDir is needed for unit tests
var TestThirdPartyManifestDir string

// OperatorConfig specifies the Verrazzano Platform Operator Config
type OperatorConfig struct {

	// The CertDir directory containing tls.crt and tls.key
	CertDir string

	// MetricsAddr is the address the metric endpoint binds to
	MetricsAddr string

	// LeaderElectionEnabled  enables/disables ensuring that there is only one active controller manager
	LeaderElectionEnabled bool

	// VersionCheckEnabled enables/disables version checking for upgrade.
	VersionCheckEnabled bool

	// RunWebhooks Runs the webhooks instead of the operator instead of the operator reconciler
	RunWebhooks bool

	// RunWebhookInit Runs the webhook init path instead of the operator reconciler
	RunWebhookInit bool

	// ResourceRequirementsValidation toggles the suppression of resource requirement validation webhook
	// default-value: false, disabling the validation
	ResourceRequirementsValidation bool

	// WebhookValidationEnabled enables/disables webhook validation without removing the webhook itself
	WebhookValidationEnabled bool

	// VerrazzanoRootDir is the root Verrazzano directory in the image
	VerrazzanoRootDir string

	// HealthCheckPeriodSeconds period for health check background task in seconds; a value of 0 disables health checks
	HealthCheckPeriodSeconds int64

	// MySQLCheckPeriodSeconds period for MySQL check background task in seconds; a value of 0 disables MySQL checks
	MySQLCheckPeriodSeconds int64

	// MySQLRepairTimeoutSeconds is the amount of time the MySQL check background thread will allow to transpire between
	// detecting a possible condition to repair, and initiating the repair logic.
	MySQLRepairTimeoutSeconds int64

	// DryRun Run installs in a dry-run mode
	DryRun bool

	// ExperimentalModules toggles the VPO to use the experimental modules feature
	ExperimentalModules bool
}

// The singleton instance of the operator config
var instance = OperatorConfig{
	CertDir:                        "/etc/webhook/certs",
	MetricsAddr:                    ":8080",
	LeaderElectionEnabled:          false,
	VersionCheckEnabled:            true,
	RunWebhookInit:                 false,
	RunWebhooks:                    false,
	ResourceRequirementsValidation: false,
	WebhookValidationEnabled:       true,
	VerrazzanoRootDir:              rootDir,
	HealthCheckPeriodSeconds:       60,
	MySQLCheckPeriodSeconds:        60,
	MySQLRepairTimeoutSeconds:      120,
	ExperimentalModules:            false,
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

// GetHelmVPOChartsDir returns the verrazzano-platform-operator helm charts dir
func GetHelmVPOChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/verrazzano-platform-operator")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmVPOChartsDirSuffix)
}

// GetHelmVMOChartsDir returns the verrazzano-monitoring-operator helm charts dir
func GetHelmVMOChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/verrazzano-monitoring-operator")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmVMOChartsDirSuffix)
}

// GetHelmAppOpChartsDir returns the Verrazzano Application Operator helm charts dir
func GetHelmAppOpChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/verrazzano-application-operator")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmAppOpChartsDirSuffix)
}

// GetHelmClusterOpChartsDir returns the Verrazzano Cluster Operator helm charts dir
func GetHelmClusterOpChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/verrazzano-cluster-operator")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmClusterOpChartsDirSuffix)
}

// GetHelmPromOpChartsDir returns the Prometheus Operator helm charts dir
func GetHelmPromOpChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/prometheus-community/kube-prometheus-stack")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmPromOpChartsDirSuffix)
}

// GetHelmKialiChartsDir returns the Kiali helm charts dir
func GetHelmKialiChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/kiali-server")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmKialiChartsDirSuffix)
}

// GetHelmOamChartsDir returns the oam-kubernetes-runtime helm charts dir
func GetHelmOamChartsDir() string {
	if TestHelmConfigDir != "" {
		return filepath.Join(TestHelmConfigDir, "/charts/oam-kubernetes-runtime")
	}
	return filepath.Join(instance.VerrazzanoRootDir, helmOamChartsDirSuffix)
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

func GetThirdPartyManifestsDir() string {
	if TestThirdPartyManifestDir != "" {
		return TestThirdPartyManifestDir
	}
	return filepath.Join(instance.VerrazzanoRootDir, thirdPartyManifestsDirSuffix)
}

// GetProfilesDir returns the profiles dir
func GetProfilesDir() string {
	if TestProfilesDir != "" {
		return TestProfilesDir
	}
	return filepath.Join(instance.VerrazzanoRootDir, profilesDirSuffix)
}

// GetProfile returns API profiles dir
func GetProfile(groupVersion schema.GroupVersion, profile string) string {
	return filepath.Join(GetProfilesDir()+"/"+groupVersion.Version, profile+".yaml")
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
	return []string{constants.VerrazzanoSystemNamespace, constants.VerrazzanoMonitoringNamespace, nginxutil.IngressNGINXNamespace(), constants.KeycloakNamespace}
}

func GetNoInjectionComponents() []string {
	return []string{"coherence-operator", "oam-kubernetes-runtime", "verrazzano-application-operator", "verrazzano-cluster-operator"}
}
