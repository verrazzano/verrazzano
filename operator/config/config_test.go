// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestConfigDefaults tests the config default values
// GIVEN a new OperatorConfig object
//  WHEN I call New
//  THEN the value returned are correct defaults
func TestConfigDefaults(t *testing.T) {
	asserts := assert.New(t)
	conf := Get()

	// The singleton instance of the operator config
	//var instance OperatorConfig = OperatorConfig{
	//	CertDir:                  "/etc/webhook/certs",
	//	InitWebhooks:             true,
	//	MetricsAddr:              ":8080",
	//	LeaderElectionEnabled:    false,
	//	VersionCheckEnabled:      true,
	//	WebhooksEnabled:          true,
	//	WebhookValidationEnabled: true,
	//	VerrazzanoRootDir:        "/verrazzano",
	//}

	asserts.Equal("/etc/webhook/certs", conf.CertDir, "CertDir is incorrect")
	asserts.True( conf.InitWebhooks, "InitWebhooks is incorrect")
	asserts.False( conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal(":8080", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.True( conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.True( conf.WebhooksEnabled, "WebhooksEnabled is incorrect")
	asserts.True( conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal("/verrazzano", conf.VerrazzanoRootDir, "VerrazzanoRootDir is incorrect")
}

// TestSetConfig tests setting config values
// GIVEN a OperatorConfig object with non-default values
//  WHEN I call Set
//  THEN Get returns the correct values
func TestSetConfig(t *testing.T) {
	asserts := assert.New(t)

	Set(OperatorConfig{
		CertDir:                  "/test/certs",
		InitWebhooks:             false,
		MetricsAddr:              "1111",
		LeaderElectionEnabled:    true,
		VersionCheckEnabled:      false,
		WebhooksEnabled:          false,
		WebhookValidationEnabled: false,
		VerrazzanoRootDir:        "/test/vz",
	})

	conf := Get()

	asserts.Equal("/test/certs", conf.CertDir, "CertDir is incorrect")
	asserts.False( conf.InitWebhooks, "InitWebhooks is incorrect")
	asserts.True( conf.LeaderElectionEnabled, "LeaderElectionEnabled is incorrect")
	asserts.Equal("1111", conf.MetricsAddr, "MetricsAddr is incorrect")
	asserts.False( conf.VersionCheckEnabled, "VersionCheckEnabled is incorrect")
	asserts.False( conf.WebhooksEnabled, "WebhooksEnabled is incorrect")
	asserts.False( conf.WebhookValidationEnabled, "WebhookValidationEnabled is incorrect")
	asserts.Equal("/test/vz", conf.VerrazzanoRootDir, "VerrazzanoRootDir is incorrect")
}
