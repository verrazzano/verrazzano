// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/stretchr/testify/assert"
)

// TestIsExternalDNSEnabledDefault tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//  WHEN the VZ config does not explicitly configure DNS
//  THEN false is returned
func TestIsExternalDNSEnabledDefault(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	assert.False(t, IsExternalDNSEnabled(vz))
}

// TestIsExternalDNSEnabledOCIDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//  WHEN the VZ config has OCI DNS configured
//  THEN true is returned
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
}

// TestIsExternalDNSEnabledWildcardDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//  WHEN the VZ config has Wildcard DNS explicitly configured
//  THEN false is returned
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
}

// TestIsExternalDNSEnabledExternalDNS tests the IsExternalDNSEnabled function
// GIVEN a call to IsExternalDNSEnabled
//  WHEN the VZ config has External DNS explicitly configured
//  THEN false is returned
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
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsKeycloakEnabled tests the IsKeycloakEnabled function
// GIVEN a call to IsKeycloakEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsConsoleEnabled tests the IsConsoleEnabled function
// GIVEN a call to IsConsoleEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &trueValue,
					},
				},
			},
		}}))
	asserts.False(IsConsoleEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &falseValue,
					},
				},
			},
		}}))
}

// TestIsFluentdEnabled tests the IsFluentdEnabled function
// GIVEN a call to IsFluentdEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsGrafanaEnabled tests the IsGrafanaEnabled function
// GIVEN a call to IsGrafanaEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &trueValue,
					},
				},
			},
		}}))
	asserts.False(IsGrafanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &falseValue,
					},
				},
			},
		}}))
}

// TestIsElasticsearchEnabled tests the IsElasticsearchEnabled function
// GIVEN a call to IsElasticsearchEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsElasticsearchEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsElasticsearchEnabled(nil))
	asserts.True(IsElasticsearchEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsElasticsearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{},
			},
		}}))
	asserts.True(IsElasticsearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &trueValue,
				},
			},
		}}))
	asserts.False(IsElasticsearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &falseValue,
				},
			},
		}}))
}

// TestIsKibanaEnabled tests the IsKibanaEnabled function
// GIVEN a call to IsKibanaEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
func TestIsKibanaEnabled(t *testing.T) {
	asserts := assert.New(t)
	asserts.True(IsKibanaEnabled(nil))
	asserts.True(IsKibanaEnabled(&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{}}))
	asserts.True(IsKibanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{},
			},
		}}))
	asserts.True(IsKibanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &trueValue,
					},
				},
			},
		}}))
	asserts.False(IsKibanaEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kibana: &vzapi.KibanaComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &falseValue,
					},
				},
			},
		}}))
}

// TestIsPrometheusEnabled tests the IsPrometheusEnabled function
// GIVEN a call to IsPrometheusEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &trueValue,
					},
				},
			},
		}}))
	asserts.False(IsPrometheusEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &falseValue,
					},
				},
			},
		}}))
}

// TestIsKialiEnabled tests the IsKialiEnabled function
// GIVEN a call to IsKialiEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsIstioEnabled tests the IsIstioEnabled function
// GIVEN a call to IsIstioEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsNGINXEnabled tests the IsNGINXEnabled function
// GIVEN a call to IsNGINXEnabled
//  THEN the value of the Enabled flag is returned if present, true otherwise (enabled by default)
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
}

// TestIsJaegerOperatorEnabled tests the IsJaegerOperatorEnabled function
// GIVEN a call to IsJaegerOperatorEnabled
//  THEN the value of the Enabled flag is returned if present, false otherwise (disabled by default)
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
}
