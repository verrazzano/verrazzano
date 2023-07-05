// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzcr

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	enabled  = true
	disabled = false
)

// TestIsExternalDNSEnabledDefault tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//
//	WHEN the VZ config does not explicitly configure DNS
//	THEN false is returned
func TestIsExternalDNSEnabledDefault(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	assert.False(t, IsExternalDNSEnabled(vz))

	vzv1beta1 := &installv1beta1.Verrazzano{}
	assert.False(t, IsExternalDNSEnabled(vzv1beta1))
}

// TestIsExternalDNSEnabledOCIDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//
//	WHEN the VZ config has OCI DNS configured
//	THEN true is returned
func TestIsExternalDNSEnabledOCIDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	assert.True(t, IsExternalDNSEnabled(vz))

	vzv1beta1 := &installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: installv1beta1.ComponentSpec{
				DNS: &installv1beta1.DNSComponent{
					OCI: &installv1beta1.OCI{
						DNSZoneName: "mydomain.com"},
				},
			},
		},
	}
	assert.True(t, IsExternalDNSEnabled(vzv1beta1))
}

// TestIsExternalDNSEnabledWildcardDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//
//	WHEN the VZ config has Wildcard DNS explicitly configured
//	THEN false is returned
func TestIsExternalDNSEnabledWildcardDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					Wildcard: &vzapi.Wildcard{
						Domain: "xip.io",
					},
				},
			},
		},
	}
	assert.False(t, IsExternalDNSEnabled(vz))

	vzv1beta := &installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: installv1beta1.ComponentSpec{
				DNS: &installv1beta1.DNSComponent{
					Wildcard: &installv1beta1.Wildcard{
						Domain: "xip.io",
					},
				},
			},
		},
	}
	assert.False(t, IsExternalDNSEnabled(vzv1beta))
}

// TestIsExternalDNSEnabledExternalDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//
//	WHEN the VZ config has External DNS explicitly configured
//	THEN false is returned
func TestIsExternalDNSEnabledExternalDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{
						Suffix: "mydomain.io",
					},
				},
			},
		},
	}
	assert.False(t, IsExternalDNSEnabled(vz))
}

var trueValue = true
var falseValue = false

// TestIsRancherEnabled tests the IsRancherEnabled function
// GIVEN a call to IsRancherEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsRancherEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsRancherEnabled(nil))
	asserts.True(IsRancherEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsRancherEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{},
			},
		}}))
	asserts.True(IsRancherEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsRancherEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsRancherEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Rancher: &installv1beta1.RancherComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsRancherEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Rancher: &installv1beta1.RancherComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsKeycloakEnabled tests the IsKeycloakEnabled function
// GIVEN a call to IsKeycloakEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsKeycloakEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsKeycloakEnabled(nil))
	asserts.True(IsKeycloakEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsKeycloakEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{},
			},
		}}))
	asserts.True(IsKeycloakEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKeycloakEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsKeycloakEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Keycloak: &installv1beta1.KeycloakComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKeycloakEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Keycloak: &installv1beta1.KeycloakComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsClusterIssuerEnabled tests the IsClusterIssuerEnabled function
// GIVEN a call to IsClusterIssuerEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsClusterIssuerEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsClusterIssuerEnabled(nil))
	asserts.True(IsClusterIssuerEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsClusterIssuerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{},
			},
		}}))
	asserts.True(IsClusterIssuerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsClusterIssuerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsClusterIssuerEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsClusterIssuerEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsCertManagerWebhookOCIEnabled tests the IsCertManagerWebhookOCIEnabled function
// GIVEN a call to IsCertManagerWebhookOCIEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsCertManagerWebhookOCIEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsCertManagerWebhookOCIEnabled(nil))
	asserts.False(IsCertManagerWebhookOCIEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsCertManagerWebhookOCIEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{},
			},
		}}))
	asserts.True(IsCertManagerWebhookOCIEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsCertManagerWebhookOCIEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsCertManagerWebhookOCIEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				CertManagerWebhookOCI: &installv1beta1.CertManagerWebhookOCIComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsCertManagerWebhookOCIEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				CertManagerWebhookOCI: &installv1beta1.CertManagerWebhookOCIComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsCertManagerWebhookOCIRequiredV1Alpha1 tests the IsCertManagerWebhookOCIRequired function
// GIVEN a call to IsCertManagerWebhookOCIRequired
//
//	THEN true is returned IF the webhook is explicitly enabled OR the issuer component is enabled and OCI DNS
//	with ACME/LetsEncrypt is configured
func TestIsCertManagerWebhookOCIRequiredV1Alpha1(t *testing.T) {
	asserts := assert.New(t)

	asserts.False(IsCertManagerWebhookOCIRequired(nil))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
			},
		}}))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{Enabled: &falseValue},
				DNS:           &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
			},
		}}))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{Enabled: &trueValue},
				DNS:           &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
			},
		}}))

	asserts.True(IsCertManagerWebhookOCIRequired(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					Enabled: &trueValue,
					IssuerConfig: vzapi.IssuerConfig{
						LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{},
					},
				},
				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
			},
		}}))

	asserts.True(IsCertManagerWebhookOCIRequired(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: vzapi.NewDefaultClusterIssuer(),
				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
					Enabled: &trueValue,
				},
			},
		}}))
}

// TestIsCertManagerWebhookOCIRequiredV1Beta1 tests the IsCertManagerWebhookOCIRequired function
// GIVEN a call to IsCertManagerWebhookOCIRequired
//
//	THEN true is returned IF the webhook is explicitly enabled OR the issuer component is enabled and OCI DNS
//	with ACME/LetsEncrypt is configured
func TestIsCertManagerWebhookOCIRequiredV1Beta1(t *testing.T) {
	asserts := assert.New(t)

	asserts.False(IsCertManagerWebhookOCIRequired(nil))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				DNS: &installv1beta1.DNSComponent{OCI: &installv1beta1.OCI{}},
			},
		}}))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{Enabled: &falseValue},
				DNS:           &installv1beta1.DNSComponent{OCI: &installv1beta1.OCI{}},
			},
		}}))

	asserts.False(IsCertManagerWebhookOCIRequired(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{Enabled: &trueValue},
				DNS:           &installv1beta1.DNSComponent{OCI: &installv1beta1.OCI{}},
			},
		}}))

	asserts.True(IsCertManagerWebhookOCIRequired(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{
					Enabled: &trueValue,
					IssuerConfig: installv1beta1.IssuerConfig{
						LetsEncrypt: &installv1beta1.LetsEncryptACMEIssuer{},
					},
				},
				DNS: &installv1beta1.DNSComponent{OCI: &installv1beta1.OCI{}},
			},
		}}))

	asserts.True(IsCertManagerWebhookOCIRequired(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: installv1beta1.NewDefaultClusterIssuer(),
				CertManagerWebhookOCI: &installv1beta1.CertManagerWebhookOCIComponent{
					Enabled: &trueValue,
				},
			},
		}}))
}

func TestIsCAConfig(t *testing.T) {
	asserts := assert.New(t)

	isCA, err := IsCAConfig(&corev1.Secret{})
	asserts.False(isCA)
	asserts.Error(err)

	asserts.False(IsCAConfig(nil))

	asserts.True(IsCAConfig(&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{}}))

	asserts.True(IsCAConfig(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{
					IssuerConfig: installv1beta1.IssuerConfig{CA: &installv1beta1.CAIssuer{}},
				},
			},
		}}))

	asserts.False(IsCAConfig(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{},
			},
		}}))

	asserts.False(IsCAConfig(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ClusterIssuer: &installv1beta1.ClusterIssuerComponent{
					IssuerConfig: installv1beta1.IssuerConfig{LetsEncrypt: &installv1beta1.LetsEncryptACMEIssuer{}},
				},
			},
		}}))

	asserts.True(IsCAConfig(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))

	asserts.True(IsCAConfig(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					IssuerConfig: vzapi.IssuerConfig{CA: &vzapi.CAIssuer{}},
				},
			},
		}}))

	asserts.False(IsCAConfig(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{},
			},
		}}))

	asserts.False(IsCAConfig(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					IssuerConfig: vzapi.IssuerConfig{LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{}},
				},
			},
		}}))
}

// TestIsConsoleEnabled tests the IsConsoleEnabled function
// GIVEN a call to IsConsoleEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsConsoleEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsConsoleEnabled(nil))
	asserts.True(IsConsoleEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsConsoleEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{},
			},
		}}))
	asserts.True(IsConsoleEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsConsoleEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsConsoleEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Console: &installv1beta1.ConsoleComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsConsoleEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Console: &installv1beta1.ConsoleComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsFluentdEnabled tests the IsFluentdEnabled function
// GIVEN a call to IsFluentdEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsFluentdEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsFluentdEnabled(nil))
	asserts.True(IsFluentdEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsFluentdEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{},
			},
		}}))
	asserts.True(IsFluentdEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentdEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsFluentdEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Fluentd: &installv1beta1.FluentdComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentdEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Fluentd: &installv1beta1.FluentdComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsGrafanaEnabled tests the IsGrafanaEnabled function
// GIVEN a call to IsGrafanaEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsGrafanaEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsGrafanaEnabled(nil))
	asserts.True(IsGrafanaEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsGrafanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{},
			},
		}}))
	asserts.True(IsGrafanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsGrafanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsGrafanaEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Grafana: &installv1beta1.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsGrafanaEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Grafana: &installv1beta1.GrafanaComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsElasticsearchEnabled tests the IsOpenSearchEnabled function
// GIVEN a call to IsOpenSearchEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsElasticsearchEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsOpenSearchEnabled(nil))
	asserts.True(IsOpenSearchEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsOpenSearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{},
			},
		}}))
	asserts.True(IsOpenSearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsOpenSearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsOpenSearchEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				OpenSearch: &installv1beta1.OpenSearchComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsOpenSearchEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				OpenSearch: &installv1beta1.OpenSearchComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsKibanaEnabled tests the IsOpenSearchDashboardsEnabled function
// GIVEN a call to IsOpenSearchDashboardsEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsKibanaEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsOpenSearchDashboardsEnabled(nil))
	asserts.True(IsOpenSearchDashboardsEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsOpenSearchDashboardsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{},
			},
		}}))
	asserts.True(IsOpenSearchDashboardsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsOpenSearchDashboardsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsOpenSearchDashboardsEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				OpenSearchDashboards: &installv1beta1.OpenSearchDashboardsComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsOpenSearchDashboardsEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				OpenSearchDashboards: &installv1beta1.OpenSearchDashboardsComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsPrometheusEnabled tests the IsPrometheusEnabled function
// GIVEN a call to IsPrometheusEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsPrometheusEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsPrometheusEnabled(nil))
	asserts.True(IsPrometheusEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsPrometheusEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{},
			},
		}}))
	asserts.True(IsPrometheusEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsPrometheusEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsPrometheusEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Prometheus: &installv1beta1.PrometheusComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsPrometheusEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Prometheus: &installv1beta1.PrometheusComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsKialiEnabled tests the IsKialiEnabled function
// GIVEN a call to IsKialiEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsKialiEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsKialiEnabled(nil))
	asserts.True(IsKialiEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsKialiEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kiali: &vzapi.KialiComponent{},
			},
		}}))
	asserts.True(IsKialiEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kiali: &vzapi.KialiComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKialiEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kiali: &vzapi.KialiComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsKialiEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Kiali: &installv1beta1.KialiComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKialiEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Kiali: &installv1beta1.KialiComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsIstioEnabled tests the IsIstioEnabled function
// GIVEN a call to IsIstioEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsIstioEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsIstioEnabled(nil))
	asserts.True(IsIstioEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsIstioEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{},
			},
		}}))
	asserts.True(IsIstioEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsIstioEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsIstioEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsIstioEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsIstioInjectionEnabled tests the IsIstioInjectionEnabled function
// GIVEN a call to IsIstioInjectionEnabled
//
//	THEN return false if either Istio is disabled OR Injection is disabled, true otherwise
func TestIsIstioInjectionEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsIstioInjectionEnabled(nil))
	asserts.True(IsIstioInjectionEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsIstioInjectionEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{},
			},
		}}))
	asserts.True(IsIstioInjectionEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsIstioInjectionEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled:          &trueValue,
					InjectionEnabled: &falseValue,
				},
			},
		}}))
	asserts.False(IsIstioInjectionEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsIstioInjectionEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsIstioInjectionEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					Enabled:          &trueValue,
					InjectionEnabled: &falseValue,
				},
			},
		}}))
	asserts.False(IsIstioInjectionEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsNGINXEnabled tests the IsNGINXEnabled function
// GIVEN a call to IsNGINXEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsNGINXEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsNGINXEnabled(nil))
	asserts.True(IsNGINXEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsNGINXEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{},
			},
		}}))
	asserts.True(IsNGINXEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsNGINXEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsNGINXEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				IngressNGINX: &installv1beta1.IngressNginxComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsNGINXEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				IngressNGINX: &installv1beta1.IngressNginxComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsJaegerOperatorEnabled tests the IsJaegerOperatorEnabled function
// GIVEN a call to IsJaegerOperatorEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsJaegerOperatorEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsJaegerOperatorEnabled(nil))
	asserts.False(IsJaegerOperatorEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsJaegerOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{},
			},
		}}))
	asserts.True(IsJaegerOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsJaegerOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsJaegerOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				JaegerOperator: &installv1beta1.JaegerOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsJaegerOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				JaegerOperator: &installv1beta1.JaegerOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsApplicationOperatorEnabled tests the IsApplicationOperatorEnabled function
// GIVEN a call to IsApplicationOperatorEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsApplicationOperatorEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsApplicationOperatorEnabled(nil))
	asserts.True(IsApplicationOperatorEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsApplicationOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ApplicationOperator: &vzapi.ApplicationOperatorComponent{},
			},
		}}))
	asserts.True(IsApplicationOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ApplicationOperator: &vzapi.ApplicationOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsApplicationOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ApplicationOperator: &vzapi.ApplicationOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsApplicationOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ApplicationOperator: &installv1beta1.ApplicationOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsApplicationOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				ApplicationOperator: &installv1beta1.ApplicationOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsVeleroEnabled tests the IsVeleroEnabled function
// GIVEN a call to IsVeleroEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsVeleroEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsVeleroEnabled(nil))
	asserts.False(IsVeleroEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsVeleroEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Velero: &vzapi.VeleroComponent{},
			},
		}}))
	asserts.True(IsVeleroEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Velero: &vzapi.VeleroComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsVeleroEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Velero: &vzapi.VeleroComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsVeleroEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Velero: &installv1beta1.VeleroComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsVeleroEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Velero: &installv1beta1.VeleroComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsCertManagerEnabled tests the IsCertManagerEnabled function
// GIVEN a call to IsCertManagerEnabled
// WHEN the CertManager component is explicitly disabled
// THEN return false, true otherwise (enabled by default)
func TestIsCertManagerEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsCertManagerEnabled(nil))
	asserts.True(IsCertManagerEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsCertManagerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{},
			},
		}}))
	asserts.True(IsCertManagerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsCertManagerEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsCertManagerEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				CertManager: &installv1beta1.CertManagerComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsCertManagerEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				CertManager: &installv1beta1.CertManagerComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsKubeStateMetricsEnabled tests the IsKubeStateMetricsEnabled function
// GIVEN a call to IsKubeStateMetricsEnabled
// WHEN the KubeStateMetrics component is explicitly enabled
// THEN return true, false otherwise (disabled by default)
func TestIsKubeStateMetricsEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsKubeStateMetricsEnabled(nil))
	asserts.False(IsKubeStateMetricsEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsKubeStateMetricsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				KubeStateMetrics: &vzapi.KubeStateMetricsComponent{},
			},
		}}))
	asserts.True(IsKubeStateMetricsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKubeStateMetricsEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsKubeStateMetricsEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				KubeStateMetrics: &installv1beta1.KubeStateMetricsComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsKubeStateMetricsEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				KubeStateMetrics: &installv1beta1.KubeStateMetricsComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsAuthProxyEnabled tests the IsAuthProxyEnabled function
// GIVEN a call to IsAuthProxyEnabled
// WHEN the AuthProxy component is explicitly disabled
// THEN return false, true otherwise (enabled by default)
func TestIsAuthProxyEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsAuthProxyEnabled(nil))
	asserts.True(IsAuthProxyEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsAuthProxyEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{},
			},
		}}))
	asserts.True(IsAuthProxyEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsAuthProxyEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsAuthProxyEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				AuthProxy: &installv1beta1.AuthProxyComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsAuthProxyEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				AuthProxy: &installv1beta1.AuthProxyComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsRancherBackupEnabled tests the IsRancherBackupEnabled function
// GIVEN a call to IsRancherBackupEnabled
// WHEN the RancherBackup component is explicitly enabled
// THEN return true, false otherwise (disabled by default)
func TestIsRancherBackupEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsRancherBackupEnabled(nil))
	asserts.False(IsRancherBackupEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsRancherBackupEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				RancherBackup: &vzapi.RancherBackupComponent{},
			},
		}}))
	asserts.True(IsRancherBackupEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				RancherBackup: &vzapi.RancherBackupComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsRancherBackupEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				RancherBackup: &vzapi.RancherBackupComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsRancherBackupEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				RancherBackup: &installv1beta1.RancherBackupComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsRancherBackupEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				RancherBackup: &installv1beta1.RancherBackupComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsPrometheusComponentsEnabled tests whether the PrometheusComponents are enabled or not
// GIVEN a call to isEnabled function of a Prometheus component
// WHEN the Prometheus component is explicitly enabled or disabled
// THEN return the value as expected in the enabled variable
func TestIsPrometheusComponentsEnabled(t *testing.T) {
	var tests = []struct {
		name      string
		cr        runtime.Object
		enabled   bool
		isEnabled func(object runtime.Object) bool
	}{
		// Prometheus Operator
		{
			"Prometheus Operator enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsPrometheusOperatorEnabled,
		},
		{
			"Prometheus Operator enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsPrometheusOperatorEnabled,
		},
		{
			"Prometheus Operator enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusOperatorEnabled,
		},
		{
			"Prometheus Operator enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusOperator: &installv1beta1.PrometheusOperatorComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusOperatorEnabled,
		},
		{
			"Prometheus Operator disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusOperatorEnabled,
		},
		{
			"Prometheus Operator disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusOperator: &installv1beta1.PrometheusOperatorComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusOperatorEnabled,
		},
		// Prometheus Adapter
		{
			"Prometheus Adapter disabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			false,
			IsPrometheusAdapterEnabled,
		},
		{
			"Prometheus Adapter disabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			false,
			IsPrometheusAdapterEnabled,
		},
		{
			"Prometheus Adapter enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusAdapter: &vzapi.PrometheusAdapterComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusAdapterEnabled,
		},
		{
			"Prometheus Adapter enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusAdapter: &installv1beta1.PrometheusAdapterComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusAdapterEnabled,
		},
		{
			"Prometheus Adapter disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusAdapter: &vzapi.PrometheusAdapterComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusAdapterEnabled,
		},
		{
			"Prometheus Adapter disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusAdapter: &installv1beta1.PrometheusAdapterComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusAdapterEnabled,
		},
		// Prometheus Pushgateway
		{
			"Prometheus Pushgateway disabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			false,
			IsPrometheusPushgatewayEnabled,
		},
		{
			"Prometheus Pushgateway disabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			false,
			IsPrometheusPushgatewayEnabled,
		},
		{
			"Prometheus Pushgateway enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusPushgatewayEnabled,
		},
		{
			"Prometheus Pushgateway enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusPushgateway: &installv1beta1.PrometheusPushgatewayComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusPushgatewayEnabled,
		},
		{
			"Prometheus Pushgateway disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusPushgatewayEnabled,
		},
		{
			"Prometheus Pushgateway disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusPushgateway: &installv1beta1.PrometheusPushgatewayComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusPushgatewayEnabled,
		},
		// Prometheus NodeExporter
		{
			"Prometheus NodeExporter enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsPrometheusNodeExporterEnabled,
		},
		{
			"Prometheus NodeExporter enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsPrometheusNodeExporterEnabled,
		},
		{
			"Prometheus NodeExporter enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusNodeExporter: &vzapi.PrometheusNodeExporterComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusNodeExporterEnabled,
		},
		{
			"Prometheus NodeExporter enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusNodeExporter: &installv1beta1.PrometheusNodeExporterComponent{Enabled: &trueValue}}}},
			true,
			IsPrometheusNodeExporterEnabled,
		},
		{
			"Prometheus NodeExporter disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{PrometheusNodeExporter: &vzapi.PrometheusNodeExporterComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusNodeExporterEnabled,
		},
		{
			"Prometheus NodeExporter disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{PrometheusNodeExporter: &installv1beta1.PrometheusNodeExporterComponent{Enabled: &falseValue}}}},
			false,
			IsPrometheusNodeExporterEnabled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.isEnabled(tt.cr))
		})
	}
}

// TestIsArgoCDEnabled tests the IsArgoCDEnabled function
// GIVEN a call to IsArgoCDEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsArgoCDEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsArgoCDEnabled(nil))
	asserts.False(IsArgoCDEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsArgoCDEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{},
			},
		}}))
	asserts.True(IsArgoCDEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsArgoCDEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

func TestIsComponentEnabled(t *testing.T) {
	var tests = []struct {
		name      string
		cr        runtime.Object
		enabled   bool
		isEnabled func(object runtime.Object) bool
	}{
		// WKO
		{
			"wko enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsWebLogicOperatorEnabled,
		},
		{
			"wko enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsWebLogicOperatorEnabled,
		},
		{
			"wko enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{WebLogicOperator: &vzapi.WebLogicOperatorComponent{Enabled: &enabled}}}},
			true,
			IsWebLogicOperatorEnabled,
		},
		{
			"wko enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{WebLogicOperator: &installv1beta1.WebLogicOperatorComponent{Enabled: &enabled}}}},
			true,
			IsWebLogicOperatorEnabled,
		},
		{
			"wko disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{WebLogicOperator: &vzapi.WebLogicOperatorComponent{Enabled: &disabled}}}},
			false,
			IsWebLogicOperatorEnabled,
		},
		{
			"wko disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{WebLogicOperator: &installv1beta1.WebLogicOperatorComponent{Enabled: &disabled}}}},
			false,
			IsWebLogicOperatorEnabled,
		},

		// COH
		{
			"coh enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsCoherenceOperatorEnabled,
		},
		{
			"coh enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsCoherenceOperatorEnabled,
		},
		{
			"coh enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{CoherenceOperator: &vzapi.CoherenceOperatorComponent{Enabled: &enabled}}}},
			true,
			IsCoherenceOperatorEnabled,
		},
		{
			"coh enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{CoherenceOperator: &installv1beta1.CoherenceOperatorComponent{Enabled: &enabled}}}},
			true,
			IsCoherenceOperatorEnabled,
		},
		{
			"coh disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{CoherenceOperator: &vzapi.CoherenceOperatorComponent{Enabled: &disabled}}}},
			false,
			IsCoherenceOperatorEnabled,
		},
		{
			"coh disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{CoherenceOperator: &installv1beta1.CoherenceOperatorComponent{Enabled: &disabled}}}},
			false,
			IsCoherenceOperatorEnabled,
		},

		// Verrazzano Component
		{
			"vz enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsVerrazzanoComponentEnabled,
		},
		{
			"vz enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsVerrazzanoComponentEnabled,
		},
		{
			"vz enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Verrazzano: &vzapi.VerrazzanoComponent{Enabled: &enabled}}}},
			true,
			IsVerrazzanoComponentEnabled,
		},
		{
			"vz enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{Verrazzano: &installv1beta1.VerrazzanoComponent{Enabled: &enabled}}}},
			true,
			IsVerrazzanoComponentEnabled,
		},
		{
			"vz disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Verrazzano: &vzapi.VerrazzanoComponent{Enabled: &disabled}}}},
			false,
			IsVerrazzanoComponentEnabled,
		},
		{
			"vz disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{Verrazzano: &installv1beta1.VerrazzanoComponent{Enabled: &disabled}}}},
			false,
			IsVerrazzanoComponentEnabled,
		},

		// OAM
		{
			"oam enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsCoherenceOperatorEnabled,
		},
		{
			"oam enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsOAMEnabled,
		},
		{
			"oam enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{OAM: &vzapi.OAMComponent{Enabled: &enabled}}}},
			true,
			IsOAMEnabled,
		},
		{
			"oam enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{OAM: &installv1beta1.OAMComponent{Enabled: &enabled}}}},
			true,
			IsOAMEnabled,
		},
		{
			"oam disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{OAM: &vzapi.OAMComponent{Enabled: &disabled}}}},
			false,
			IsOAMEnabled,
		},
		{
			"oam disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{OAM: &installv1beta1.OAMComponent{Enabled: &disabled}}}},
			false,
			IsOAMEnabled,
		},

		// MySQL Operator
		{
			"mysqlop enabled when empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
			IsMySQLOperatorEnabled,
		},
		{
			"mysqlop enabled when empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
			IsMySQLOperatorEnabled,
		},
		{
			"mysqlop enabled when component enabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{MySQLOperator: &vzapi.MySQLOperatorComponent{Enabled: &enabled}}}},
			true,
			IsMySQLOperatorEnabled,
		},
		{
			"mysqlop enabled when component enabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{MySQLOperator: &installv1beta1.MySQLOperatorComponent{Enabled: &enabled}}}},
			true,
			IsMySQLOperatorEnabled,
		},
		{
			"mysqlop disabled when component disabled, v1alpha1 CR",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{MySQLOperator: &vzapi.MySQLOperatorComponent{Enabled: &disabled}}}},
			false,
			IsMySQLOperatorEnabled,
		},
		{
			"mysqlop disabled when component disabled, v1beta1 CR",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{Components: installv1beta1.ComponentSpec{MySQLOperator: &installv1beta1.MySQLOperatorComponent{Enabled: &disabled}}}},
			false,
			IsMySQLOperatorEnabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.isEnabled(tt.cr))
		})
	}
}

// TestIsFluentOperatorEnabled tests the IsFluentOperatorEnabled function
// GIVEN a call to IsFluentOperatorEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
func TestIsFluentOperatorEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsFluentOperatorEnabled(nil))
	asserts.False(IsFluentOperatorEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsFluentOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentOperator: &vzapi.FluentOperatorComponent{},
			},
		}}))
	asserts.True(IsFluentOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentOperator: &vzapi.FluentOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentOperatorEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentOperator: &vzapi.FluentOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsFluentOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				FluentOperator: &installv1beta1.FluentOperatorComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentOperatorEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				FluentOperator: &installv1beta1.FluentOperatorComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsFluentbitOpensearchOutputEnabled tests the IsFluentbitOpensearchOutputEnabled function
// GIVEN a call to IsFluentbitOpensearchOutputEnabled
//
//	THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default).
func TestIsFluentbitOpensearchOutputEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(IsFluentbitOpensearchOutputEnabled(nil))
	asserts.False(IsFluentbitOpensearchOutputEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.False(IsFluentbitOpensearchOutputEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentbitOpensearchOutput: &vzapi.FluentbitOpensearchOutputComponent{},
			},
		}}))
	asserts.True(IsFluentbitOpensearchOutputEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentbitOpensearchOutput: &vzapi.FluentbitOpensearchOutputComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentbitOpensearchOutputEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				FluentbitOpensearchOutput: &vzapi.FluentbitOpensearchOutputComponent{
					Enabled: &falseValue,
				},
			},
		}}))
	asserts.True(IsFluentbitOpensearchOutputEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				FluentbitOpensearchOutput: &installv1beta1.FluentbitOpensearchOutputComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsFluentbitOpensearchOutputEnabled(
		&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				FluentbitOpensearchOutput: &installv1beta1.FluentbitOpensearchOutputComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsVMOEnabled tests the IsVMOEnabled function
// GIVEN a call to IsVMOEnabled
//
//	THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsVMOEnabled(t *testing.T) {
	var tests = []struct {
		name    string
		cr      runtime.Object
		enabled bool
	}{
		{
			"enabled on nil CR",
			nil,
			true,
		},
		{
			"enabled on empty v1alpha1 CR",
			&vzapi.Verrazzano{},
			true,
		},
		{
			"enabled on v1alpha1 CR with Prometheus disabled",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Prometheus: &vzapi.PrometheusComponent{Enabled: &falseValue},
				}}},
			true,
		},
		{
			"enabled on v1alpha1 CR with Prometheus Operator disabled",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
				}}},
			true,
		},
		{
			"enabled on v1alpha1 CR with Opensearch and Opensearch dashboards disabled",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
					Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
				}}},
			true,
		},
		{
			"disabled on v1alpha1 CR with OpenSearch, OpenSearchDashboards and Grafana disabled",
			&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
					Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
					Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
				}}},
			false,
		},
		{
			"enabled on empty v1beta1 CR",
			&installv1beta1.Verrazzano{},
			true,
		},
		{
			"enabled on v1beta1 CR with Prometheus disabled",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
				Components: installv1beta1.ComponentSpec{
					Prometheus: &installv1beta1.PrometheusComponent{Enabled: &falseValue},
				}}},
			true,
		},
		{
			"disabled on v1beta1 CR with OpenSearch, OpenSearchDashboards and Grafana disabled",
			&installv1beta1.Verrazzano{Spec: installv1beta1.VerrazzanoSpec{
				Components: installv1beta1.ComponentSpec{
					OpenSearch:           &installv1beta1.OpenSearchComponent{Enabled: &falseValue},
					OpenSearchDashboards: &installv1beta1.OpenSearchDashboardsComponent{Enabled: &falseValue},
					Grafana:              &installv1beta1.GrafanaComponent{Enabled: &falseValue},
				}}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, IsVMOEnabled(tt.cr))
		})
	}
}
