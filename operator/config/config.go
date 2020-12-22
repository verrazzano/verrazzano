// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

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

	// The root directory of the verrazzano files in the image
	VerrazzanoRootDir string
}

// The singleton instance of the operator config
var instance OperatorConfig = OperatorConfig{
	CertDir:                  "/etc/webhook/certs",
	InitWebhooks:             true,
	MetricsAddr:              ":8080",
	LeaderElectionEnabled:    false,
	VersionCheckEnabled:      true,
	WebhooksEnabled:          true,
	WebhookValidationEnabled: true,
	VerrazzanoRootDir:        "/verrazzano",
}

// Set saves the operator config.  This should only be called at operator startup and during unit tests
func Set(config OperatorConfig) {
	instance = config
}

// Get returns the singleton instance of the operator config
func Get() OperatorConfig {
	return instance
}
