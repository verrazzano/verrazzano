// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"fmt"
	"testing"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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

// TestFindVolumeTemplate Test the FindVolumeTemplate utility function
// GIVEN a call to FindVolumeTemplate
// WHEN valid or invalid arguments are given
// THEN true and the found template are is returned if found, nil/false otherwise
func TestFindVolumeTemplate(t *testing.T) {

	specTemplateList := []vzapi.VolumeClaimSpecTemplate{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "defVolume",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "template1"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "temp1Volume",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "template2"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "temp2Volume",
			},
		},
	}
	// Test boundary conditions
	invalidName, found := FindVolumeTemplate("blah", specTemplateList)
	assert.Nil(t, invalidName)
	assert.False(t, found)
	emptyName, found2 := FindVolumeTemplate("", specTemplateList)
	assert.Nil(t, emptyName)
	assert.False(t, found2)
	nilList, found3 := FindVolumeTemplate("default", nil)
	assert.Nil(t, nilList)
	assert.False(t, found3)
	emptyList, found4 := FindVolumeTemplate("default", []vzapi.VolumeClaimSpecTemplate{})
	assert.Nil(t, emptyList)
	assert.False(t, found4)

	// Test normal behavior
	defTemplate, found := FindVolumeTemplate("default", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "defVolume", defTemplate.VolumeName)
	temp1, found := FindVolumeTemplate("template1", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "temp1Volume", temp1.VolumeName)
	temp2, found := FindVolumeTemplate("template2", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "temp2Volume", temp2.VolumeName)

}
