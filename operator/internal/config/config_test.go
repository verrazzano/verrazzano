// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigDefaults tests the config default values
// GIVEN a new OperatorConfig object
//  WHEN I call New
//  THEN the value returned are correct defaults
func TestConfigDefaults(t *testing.T) {
	asserts := assert.New(t)
	conf := Get()

	asserts.Equal("/etc/webhook/certs", conf.CertDir, "CertDir is incorrect")
	asserts.False(conf.InitWebhooks, "InitWebhooks is incorrect")
	asserts.False(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal(":8080", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.True(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.True(conf.WebhooksEnabled, "WebhooksEnabled is incorrect")
	asserts.True(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal("/verrazzano/operator/scripts/install", conf.VerrazzanoInstallDir, "VerrazzanoInstallDir is incorrect")
	asserts.Equal("/verrazzano/thirdparty/charts", conf.ThirdpartyChartsDir, "ThirdpartyChartsDir is incorrect")
}

// TestSetConfig tests setting config values
// GIVEN an OperatorConfig object with non-default values
//  WHEN I call Set
//  THEN Get returns the correct values
func TestSetConfig(t *testing.T) {
	asserts := assert.New(t)

	Set(OperatorConfig{
		CertDir:                  "/test/certs",
		InitWebhooks:             true,
		MetricsAddr:              "1111",
		LeaderElectionEnabled:    true,
		VersionCheckEnabled:      false,
		WebhooksEnabled:          false,
		WebhookValidationEnabled: false,
		VerrazzanoInstallDir:     "/test/vz",
		ThirdpartyChartsDir:      "/test/thirdparty",
	})

	conf := Get()

	asserts.Equal("/test/certs", conf.CertDir, "CertDir is incorrect")
	asserts.True(conf.InitWebhooks, "InitWebhooks is incorrect")
	asserts.True(conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal("1111", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.False(conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.False(conf.WebhooksEnabled, "WebhooksEnabled is incorrect")
	asserts.False(conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal("/test/vz", conf.VerrazzanoInstallDir, "VerrazzanoInstallDir is incorrect")
	asserts.Equal("/test/thirdparty", conf.ThirdpartyChartsDir, "ThirdpartyChartsDir is incorrect")
}
