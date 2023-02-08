// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestConfigDefaults tests the config default values
// GIVEN a new OperatorConfig object
//
//	WHEN I call New
//	THEN the value returned are correct defaults
func TestConfigDefaults(t *testing.T) {
	asserts := assert.New(t)
	conf := Get()
	asserts.Equal("/etc/webhook/certs", conf.CertDir, "CertDir is incorrect")
	asserts.False(conf.RunWebhookInit, "RunWebhookInit is incorrect")
	asserts.False(conf.RunWebhooks, "RunWebhooks is correct")
	asserts.Equal(conf.DryRun, false, "Default dry run is false")
	asserts.False(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal(":8080", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.Equal(int64(60), conf.HealthCheckPeriodSeconds, "Default health check period is correct")
	asserts.Equal(int64(60), conf.MySQLCheckPeriodSeconds, "Default MySQL check period is correct")
	asserts.Equal(int64(120), conf.MySQLRepairTimeoutSeconds, "Default MySQL repair timeout is correct")
	asserts.True(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.False(conf.RunWebhooks, "RunWebhooks is incorrect")
	asserts.False(conf.ResourceRequirementsValidation, "ResourceRequirementsValidation default value is incorrect")
	asserts.True(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal(conf.VerrazzanoRootDir, "/verrazzano", "VerrazzanoRootDir is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config", GetHelmConfigDir(), "GetHelmConfigDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts", GetHelmChartsDir(), "GetHelmChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts/verrazzano-monitoring-operator", GetHelmVMOChartsDir(), "GetHelmVmoChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts/verrazzano-application-operator", GetHelmAppOpChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts/verrazzano-cluster-operator", GetHelmClusterOpChartsDir(), "GetHelmClusterOpChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts/kiali-server", GetHelmKialiChartsDir(), "GetHelmKialiChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts/oam-kubernetes-runtime", GetHelmOamChartsDir(), "GetHelmOamChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/overrides", GetHelmOverridesDir(), "GetHelmOverridesDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/scripts/install", GetInstallDir(), "GetInstallDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator", GetPlatformDir(), "GetPlatformDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts", GetThirdPartyDir(), "GetThirdPartyDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/manifests/profiles", GetProfilesDir(), "GetProfilesDir() is correct")
	asserts.Equal("/verrazzano/platform-operator/helm_config", GetHelmConfigDir(), "GetHelmConfigDir() is correct")
	asserts.Equal("/verrazzano/platform-operator/verrazzano-bom.json", GetDefaultBOMFilePath(), "GetDefaultBOMFilePath() is correct")
}

// TestSetConfig tests setting config values
// GIVEN an OperatorConfig object with non-default values
//
//		WHEN I call Set
//		THEN Get returns the correct values
//	    Able to override variables
func TestSetConfig(t *testing.T) {
	asserts := assert.New(t)
	vzsystemNamespace := []string{"verrazzano-system", "verrazzano-monitoring", "ingress-nginx", "keycloak"}
	vznonsystemNamespace := []string{"coherence-operator", "oam-kubernetes-runtime", "verrazzano-application-operator", "verrazzano-cluster-operator"}
	TestHelmConfigDir = "/etc/verrazzano/helm_config"
	TestProfilesDir = "/etc/verrazzano/profile"
	bomFilePathOverride = "/etc/verrazzano/bom.json"
	groupVersion := schema.GroupVersion{
		Group:   "test",
		Version: "1.0",
	}
	Set(OperatorConfig{
		CertDir:                        "/test/certs",
		RunWebhookInit:                 true,
		MetricsAddr:                    "1111",
		LeaderElectionEnabled:          true,
		VersionCheckEnabled:            false,
		RunWebhooks:                    true,
		ResourceRequirementsValidation: true,
		WebhookValidationEnabled:       false,
		VerrazzanoRootDir:              "/root",
		HealthCheckPeriodSeconds:       int64(0),
		MySQLCheckPeriodSeconds:        int64(0),
		DryRun:                         true,
	})

	conf := Get()
	SetDefaultBomFilePath("/etc/bom.json")
	defer SetDefaultBomFilePath("")
	asserts.Equal("/test/certs", conf.CertDir, "CertDir is incorrect")
	asserts.True(conf.RunWebhookInit, "RunWebhookInit is incorrect")
	asserts.True(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal("1111", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.False(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.True(conf.RunWebhooks, "RunWebhooks is incorrect")
	asserts.True(conf.ResourceRequirementsValidation, "ResourceRequirementsValidation default value is incorrect")
	asserts.False(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal(conf.DryRun, true, "Default dry run is true")
	asserts.Equal("/root", conf.VerrazzanoRootDir, "VerrazzanoRootDir is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config", GetHelmConfigDir(), "GetHelmConfigDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts", GetHelmChartsDir(), "GetHelmChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/verrazzano-monitoring-operator", GetHelmVMOChartsDir(), "GetHelmVmoChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/verrazzano-application-operator", GetHelmAppOpChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/verrazzano-cluster-operator", GetHelmClusterOpChartsDir(), "GetHelmClusterOpChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/overrides", GetHelmOverridesDir(), "GetHelmOverridesDir() is incorrect")
	asserts.Equal("/root/platform-operator/scripts/install", GetInstallDir(), "GetInstallDir() is incorrect")
	asserts.Equal("/root/platform-operator", GetPlatformDir(), "GetPlatformDir() is incorrect")
	asserts.Equal("/root/platform-operator/thirdparty/charts", GetThirdPartyDir(), "GetThirdPartyDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/prometheus-community/kube-prometheus-stack", GetHelmPromOpChartsDir(), "GetHelmPromOpChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/kiali-server", GetHelmKialiChartsDir(), "GetHelmKialiChartsDir() is incorrect")
	asserts.Equal("/etc/verrazzano/helm_config/charts/oam-kubernetes-runtime", GetHelmOamChartsDir(), "GetHelmOamChartsDir() is incorrect")
	asserts.Equal("/root/platform-operator/thirdparty/manifests", GetThirdPartyManifestsDir(), "GetThirdPartyManifestsDir is incorrect")
	asserts.Equal("/etc/verrazzano/profile", GetProfilesDir(), "GetProfilesDir() is incorrect")
	asserts.Equal("/etc/bom.json", GetDefaultBOMFilePath(), "GetDefaultBOMFilePath() is incorrect")
	asserts.Equal(vzsystemNamespace, GetInjectedSystemNamespaces(), "GetInjectedSystemNamespaces() is incorrect")
	asserts.Equal(vznonsystemNamespace, GetNoInjectionComponents(), "GetNoInjectionComponents() is incorrect")
	asserts.Equal("/etc/verrazzano/profile/1.0/dev.yaml", GetProfile(groupVersion, "dev"), "GetProfile() is correct")
}
