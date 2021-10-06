// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package nginx

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestAppendNGINXOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass a VZ spec with defaults
//  THEN the values created properly
func TestAppendNGINXOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestAppendNGINXOverridesWithInstallArgs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass in extra NGINX install args
//  THEN the values are translated properly
func TestAppendNGINXOverridesWithInstallArgs(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					NGINXInstallArgs: []vzapi.InstallArgs{
						{Name: "key", Value: "value"},
						{Name: "listKey", ValueList: []string{"value1", "value2"}},
					},
				},
			},
		},
	}
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
}

// TestAppendNGINXOverridesExtraKVs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass in a KeyValue list
//  THEN the values passed in are preserved and no errors occur
func TestAppendNGINXOverridesWithExternalDNS(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName:            "myzone",
						DNSZoneCompartmentOCID: "myocid",
					},
				},
				Ingress: &vzapi.IngressNginxComponent{
					NGINXInstallArgs: []vzapi.InstallArgs{
						{Name: "key", Value: "value"},
						{Name: "listKey", ValueList: []string{"value1", "value2"}},
					},
				},
			},
		},
	}
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 6)
}

// TestAppendNGINXOverridesExtraKVs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass in a KeyValue list
//  THEN the values passed in are preserved and no errors occur
func TestAppendNGINXOverridesExtraKVs(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
		},
	}
	kvs := []bom.KeyValue{
		{Key: "Key", Value: "Value"},
	}
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, vz, false), ComponentName, ComponentNamespace, "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
}

// TestNGINXPreInstall tests the PreInstall fn
// GIVEN a call to this fn
//  WHEN I call PreInstall
//  THEN no errors are returned
func TestNGINXPreInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := PreInstall(spi.NewContext(zap.S(), client, &vzapi.Verrazzano{}, false), ComponentName, ComponentNamespace, "")
	assert.NoError(t, err)
}

// TestIsNGINXReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsNGINXReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ControllerName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      BackendName,
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
			},
		},
	)
	assert.True(t, IsReady(spi.NewContext(zap.S(), fakeClient, nil, false), ComponentName, ComponentNamespace))
}

// TestIsNGINXNotReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsNGINXNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ControllerName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      BackendName,
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UnavailableReplicas: 1,
			},
		},
	)
	assert.False(t, IsReady(spi.NewContext(zap.S(), fakeClient, nil, false), "", constants.VerrazzanoSystemNamespace))
}

// TestPostInstallWithPorts tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the VZ ingress has port overrides configured
//  THEN no error is returned
func TestPostInstallWithPorts(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:     "overrideport",
							Protocol: "tcp",
							Port:     1000,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2000,
							},
						},
					},
				},
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: ControllerName, Namespace: ComponentNamespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "duplicatePort",
					Protocol: "tcp",
					Port:     1000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 2000,
					},
				},
				{
					Name:     "additionalPort",
					Protocol: "tcp",
					Port:     1000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 2000,
					},
				},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, svc)
	err := PostInstall(spi.NewContext(zap.S(), fakeClient, vz, false), ComponentName, ComponentNamespace)
	assert.NoError(t, err)
}

// TestPostInstallNoPorts tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the VZ ingress has no port overrides configured
//  THEN no error is returned
func TestPostInstallNoPorts(t *testing.T) {
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
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := PostInstall(spi.NewContext(zap.S(), fakeClient, vz, false), ComponentName, ComponentNamespace)
	assert.NoError(t, err)
}

// TestPostInstallDryRun tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the context DryRun flag is true
//  THEN no error is returned
func TestPostInstallDryRun(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := PostInstall(spi.NewContext(zap.S(), fakeClient, nil, true), ComponentName, ComponentNamespace)
	assert.NoError(t, err)
}

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
			Namespace: ComponentNamespace,
			Name:      ControllerName,
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
			Namespace: ComponentNamespace,
			Name:      ControllerName,
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
	const expectedIP = "11.22.33.44"
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ControllerName,
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
			Namespace: ComponentNamespace,
			Name:      ControllerName,
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
			Namespace: ComponentNamespace,
			Name:      ControllerName,
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

// TestNewComponent tests the NewComponent function
// GIVEN a call to NewComponent
//  THEN the NGINX component is returned
func TestNewComponent(t *testing.T) {
	component := NewComponent()
	assert.NotNil(t, component)
	assert.Equal(t, ComponentName, component.Name())
}
