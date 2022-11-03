// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	asserts.False(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal(":8080", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.Equal(int64(60), conf.HealthCheckPeriodSeconds, "Default health check period is correct")
	asserts.True(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.False(conf.RunWebhooks, "RunWebhooks is incorrect")
	asserts.True(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal(conf.VerrazzanoRootDir, "/verrazzano", "VerrazzanoRootDir is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config", GetHelmConfigDir(), "GetHelmConfigDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts", GetHelmChartsDir(), "GetHelmChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts/verrazzano-monitoring-operator", GetHelmVMOChartsDir(), "GetHelmVmoChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/charts/verrazzano-application-operator", GetHelmAppOpChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts/kiali-server", GetHelmKialiChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts/oam-kubernetes-runtime", GetHelmOamChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/helm_config/overrides", GetHelmOverridesDir(), "GetHelmOverridesDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/scripts/install", GetInstallDir(), "GetInstallDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator", GetPlatformDir(), "GetPlatformDir() is incorrect")
	asserts.Equal("/verrazzano/platform-operator/thirdparty/charts", GetThirdPartyDir(), "GetThirdPartyDir() is incorrect")
}

// TestSetConfig tests setting config values
// GIVEN an OperatorConfig object with non-default values
//
//	WHEN I call Set
//	THEN Get returns the correct values
func TestSetConfig(t *testing.T) {
	asserts := assert.New(t)

	Set(OperatorConfig{
		CertDir:                  "/test/certs",
		RunWebhookInit:           true,
		MetricsAddr:              "1111",
		LeaderElectionEnabled:    true,
		VersionCheckEnabled:      false,
		RunWebhooks:              true,
		WebhookValidationEnabled: false,
		VerrazzanoRootDir:        "/root",
		HealthCheckPeriodSeconds: int64(0),
	})

	conf := Get()

	asserts.Equal("/test/certs", conf.CertDir, "CertDir is incorrect")
	asserts.True(conf.RunWebhookInit, "RunWebhookInit is incorrect")
	asserts.True(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal("1111", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.False(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.True(conf.RunWebhooks, "RunWebhooks is incorrect")
	asserts.False(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal("/root", conf.VerrazzanoRootDir, "VerrazzanoRootDir is incorrect")
	asserts.Equal("/root/platform-operator/helm_config", GetHelmConfigDir(), "GetHelmConfigDir() is incorrect")
	asserts.Equal("/root/platform-operator/helm_config/charts", GetHelmChartsDir(), "GetHelmChartsDir() is incorrect")
	asserts.Equal("/root/platform-operator/helm_config/charts/verrazzano-monitoring-operator", GetHelmVMOChartsDir(), "GetHelmVmoChartsDir() is incorrect")
	asserts.Equal("/root/platform-operator/helm_config/charts/verrazzano-application-operator", GetHelmAppOpChartsDir(), "GetHelmAppOpChartsDir() is incorrect")
	asserts.Equal("/root/platform-operator/helm_config/overrides", GetHelmOverridesDir(), "GetHelmOverridesDir() is incorrect")
	asserts.Equal("/root/platform-operator/scripts/install", GetInstallDir(), "GetInstallDir() is incorrect")
	asserts.Equal("/root/platform-operator", GetPlatformDir(), "GetPlatformDir() is incorrect")
	asserts.Equal("/root/platform-operator/thirdparty/charts", GetThirdPartyDir(), "GetThirdPartyDir() is incorrect")
}
