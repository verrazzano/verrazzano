package config

// Config specfies the Verrazzano Platform Operator Config
type OperatorConfig struct {

	// The CertDir directory containing tls.crt and tls.key
	CertDir string

	// MetricsAddr is the address the metric endpoint binds to
	MetricsAddr string

	// EnableLeaderElection ensures there is only one active controller manager
	EnableLeaderElection bool

	// EnableWebhooks enables Webhooks for the operator
	EnableWebhooks bool

	// EnableWebhookValidation allows disabling webhook validation without removing the webhook itself
	EnableWebhookValidation bool

	// InitWebhooks enables initialzation of webhooks for the operator
	InitWebhooks bool
}

// The singleton instance of the operator config
var instance OperatorConfig = OperatorConfig{}

// Instance returns the singleton instance of the operator config
func Instance() OperatorConfig {
	return instance
}
