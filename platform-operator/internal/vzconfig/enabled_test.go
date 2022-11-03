// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/stretchr/testify/assert"
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
