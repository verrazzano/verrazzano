// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test_getServiceTypeLoadBalancer tests the GetServiceType function
// GIVEN a call to GetServiceType
//  WHEN the Ingress specifies a LoadBalancer type
//  THEN the LoadBalancer type is returned with no error
func Test_getServiceTypeLoadBalancer(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}

	svcType, err := GetServiceType(vz)
	assert.NoError(t, err)
	assert.Equal(t, vzapi.LoadBalancer, svcType)
}

// Test_getServiceTypeNodePort tests the GetServiceType function
// GIVEN a call to GetServiceType
//  WHEN the Ingress specifies a NodePort type
//  THEN the NodePort type is returned with no error
func Test_getServiceTypeNodePort(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.NodePort,
				},
			},
		},
	}

	svcType, err := GetServiceType(vz)
	assert.NoError(t, err)
	assert.Equal(t, vzapi.NodePort, svcType)
}

// Test_getServiceTypeInvalidType tests the GetServiceType function
// GIVEN a call to GetServiceType
//  WHEN the Ingress specifies invalid service type
//  THEN an empty string and an error are returned
func Test_getServiceTypeInvalidType(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: "somethingbad",
				},
			},
		},
	}

	svcType, err := GetServiceType(vz)
	assert.Error(t, err)
	assert.Equal(t, vzapi.IngressType(""), svcType)
}

// TestGetIngressLoadBalancerIPServiceNotFound tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a LB type and no IP is found
//  THEN an error is returned
func TestGetIngressLoadBalancerIPServiceNotFound(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	_, err := GetIngressIP(fakeClient, vz)
	assert.Error(t, err)
}

// TestGetIngressLoadBalancerIP tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a LB type the LoadBalancerStatus has an IP
//  THEN the IP and no error are returned
func TestGetIngressLoadBalancerIP(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	const expectedIP = "11.22.33.44"
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.IngressNamespace,
			Name:      vpoconst.NGINXControllerServiceName,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{IP: expectedIP},
				},
			},
		},
	})
	ip, err := GetIngressIP(fakeClient, vz)
	assert.NoError(t, err)
	assert.Equal(t, expectedIP, ip)
}

// TestGetIngressExternalIP tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a LB type the service spec has an ExternalIP
//  THEN the ExternalIP and no error are returned
func TestGetIngressExternalIP(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	const expectedIP = "11.22.33.44"
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.IngressNamespace,
			Name:      vpoconst.NGINXControllerServiceName,
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{"11.22.33.44"},
		},
	})
	ip, err := GetIngressIP(fakeClient, vz)
	assert.NoError(t, err)
	assert.Equal(t, expectedIP, ip)
}

// TestGetIngressNodePortIP tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a NodePort type
//  THEN the loopback address and no error are returned
func TestGetIngressNodePortIP(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.NodePort,
				},
			},
		},
	}
	const expectedIP = "127.0.0.1"
	ip, err := GetIngressIP(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.NoError(t, err)
	assert.Equal(t, expectedIP, ip)
}

// TestGetIngressLoadBalancerNoAddressFound tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a LB type and no IP is found
//  THEN an error is returned
func TestGetIngressLoadBalancerNoAddressFound(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.IngressNamespace,
			Name:      vpoconst.NGINXControllerServiceName,
		},
	})
	_, err := GetIngressIP(fakeClient, vz)
	assert.Error(t, err)
}

// TestGetDNSSuffixDefaultWildCardLoadBalancer tests the GetDNSSuffix function
// GIVEN a call to GetDNSSuffix
//  WHEN the VZ config Ingress is a LB type with a valid IP found and no DNS is configured
//  THEN the default wildcard domain for the LB service is returned
func TestGetDNSSuffixDefaultWildCardLoadBalancer(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	const expectedIP = "11.22.33.44"
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.IngressNamespace,
			Name:      vpoconst.NGINXControllerServiceName,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{IP: expectedIP},
				},
			},
		},
	})
	dnsDomain, err := GetDNSSuffix(fakeClient, vz)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s.nip.io", expectedIP), dnsDomain)
}

// TestGetDNSSuffixWildCardLoadBalancer tests the GetDNSSuffix function
// GIVEN a call to GetDNSSuffix
//  WHEN the VZ config Ingress is a LB type with a valid IP found a non-default Wildcard DNS specified
//  THEN the valid wildcard domain for the LB service is returned
func TestGetDNSSuffixWildCardLoadBalancer(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
				DNS: &vzapi.DNSComponent{
					Wildcard: &vzapi.Wildcard{
						Domain: "xip.io",
					},
				},
			},
		},
	}
	const expectedIP = "11.22.33.44"
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.IngressNamespace,
			Name:      vpoconst.NGINXControllerServiceName,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{IP: expectedIP},
				},
			},
		},
	})
	dnsDomain, err := GetDNSSuffix(fakeClient, vz)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s.xip.io", expectedIP), dnsDomain)
}

// TestGetDNSSuffixOCIDNS tests the GetDNSSuffix function
// GIVEN a call to GetDNSSuffix
//  WHEN the VZ config Ingress has OCI DNS configured
//  THEN the correct OCI DNS domain is returned
func TestGetDNSSuffixOCIDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	dnsDomain, err := GetDNSSuffix(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.NoError(t, err)
	assert.Equal(t, "mydomain.com", dnsDomain)
}

// TestGetDNSSuffixExternalDNS tests the GetDNSSuffix function
// GIVEN a call to GetDNSSuffix
//  WHEN the VZ config Ingress has External DNS configured
//  THEN the correct External DNS domain is returned
func TestGetDNSSuffixExternalDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{
						Suffix: "mydomain.com",
					},
				},
			},
		},
	}
	dnsDomain, err := GetDNSSuffix(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.NoError(t, err)
	assert.Equal(t, "mydomain.com", dnsDomain)
}

// TestGetDNSSuffixNoSuffix tests the GetDNSSuffix function
// GIVEN a call to GetDNSSuffix
//  WHEN the VZ config Ingress has External DNS configured with an empty domain
//  THEN an error is returned
func TestGetDNSSuffixNoSuffix(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{
						Suffix: "",
					},
				},
			},
		},
	}
	_, err := GetDNSSuffix(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.Error(t, err)
}

// TestGetEnvName tests the GetEnvName function
// GIVEN a call to GetEnvName
//  WHEN the VZ config specifies an env name
//  THEN the configured env name is returned
func TestGetEnvName(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
		},
	}
	assert.Equal(t, "myenv", GetEnvName(vz))
}

// TestGetEnvNameDefault tests the GetEnvName function
// GIVEN a call to GetEnvName
//  WHEN the VZ config does not explicitly configure an EnvironmentName
//  THEN then "default" is returned
func TestGetEnvNameDefault(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	assert.Equal(t, "default", GetEnvName(vz))
}

// TestBuildDNSDomainDefaultEnv tests the BuildDNSDomain function
// GIVEN a call to BuildDNSDomain
//  WHEN the VZ config specifies no env name
//  THEN the domain name is correctly returned
func TestBuildDNSDomainDefaultEnv(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	domain, err := BuildDNSDomain(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.NoError(t, err)
	assert.Equal(t, "default.mydomain.com", domain)
}

// TestBuildDNSDomainCustomEnv tests the BuildDNSDomain function
// GIVEN a call to BuildDNSDomain
//  WHEN the VZ config specifies a custom env name
//  THEN the domain name is correctly returned
func TestBuildDNSDomainCustomEnv(t *testing.T) {
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
	domain, err := BuildDNSDomain(fake.NewFakeClientWithScheme(k8scheme.Scheme), vz)
	assert.NoError(t, err)
	assert.Equal(t, "myenv.mydomain.com", domain)
}

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

// TestFindVolumeTemplate tests the FindVolumeTemplate function
// GIVEN a call to FindVolumeTemplate
//  WHEN the template list does and does not have the requested template by name
//  THEN nil/false is returned if not found, true/template are returned otherwise
func TestFindVolumeTemplate(t *testing.T) {
	template, found := FindVolumeTemplate("vmi",
		[]vzapi.VolumeClaimSpecTemplate{
			{ObjectMeta: metav1.ObjectMeta{Name: "vmi"}},
		})
	assert.True(t, found)
	assert.NotNil(t, template)

	template, found = FindVolumeTemplate("boo",
		[]vzapi.VolumeClaimSpecTemplate{
			{ObjectMeta: metav1.ObjectMeta{Name: "vmi"}},
		})
	assert.False(t, found)
	assert.Nil(t, template)
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
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &trueValue,
					},
				},
			},
		}}))
	asserts.False(IsElasticsearchEnabled(
		&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &falseValue,
					},
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
	//asserts.True(IsIstioEnabled(
	//	&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
	//		Components: vzapi.ComponentSpec{
	//			Istio: &vzapi.IstioComponent{
	//				Enabled: &trueValue,
	//			},
	//		},
	//	}}))
	//asserts.False(IsIstioEnabled(
	//	&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
	//		Components: vzapi.ComponentSpec{
	//			Istio: &vzapi.IstioComponent{
	//				Enabled: &falseValue,
	//			},
	//		},
	//	}}))
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
	//asserts.True(IsNGINXEnabled(
	//	&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
	//		Components: vzapi.ComponentSpec{
	//			Ingress: &vzapi.IngressNginxComponent{
	//				Enabled: &trueValue,
	//			},
	//		},
	//	}}))
	//asserts.False(IsNGINXEnabled(
	//	&vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{
	//		Components: vzapi.ComponentSpec{
	//			Ingress: &vzapi.IngressNginxComponent{
	//				Enabled: &falseValue,
	//			},
	//		},
	//	}}))
}
