// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/test/ip"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testDomain = "mydomain.com"

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

	svcType, err := GetIngressServiceType(vz)
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

	svcType, err := GetIngressServiceType(vz)
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

	svcType, err := GetIngressServiceType(vz)
	assert.Error(t, err)
	assert.Equal(t, vzapi.IngressType(""), svcType)
}

// TestGetIngressServiceNotFound tests the GetIngressIP function
// GIVEN a call to GetIngressIP
//  WHEN the VZ config Ingress is a LB type and no service is found
//  THEN an error is returned
func TestGetIngressServiceNotFound(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					Type: vzapi.LoadBalancer,
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	_, err := GetIngressIP(fakeClient, vz)
	assert.Error(t, err)
}

func TestGetIngressIP(t *testing.T) {
	testExternalIP := ip.RandomIP()
	testLoadBalancerIP := ip.RandomIP()
	tests := []struct {
		name        string
		serviceType vzapi.IngressType
		lbIP        string
		externalIP  string
		want        string
		wantErr     bool
	}{
		{
			name:        "lb no address found",
			serviceType: vzapi.LoadBalancer,
			wantErr:     true,
		},
		{
			name:        "lb with both lb and external ip",
			serviceType: vzapi.LoadBalancer,
			lbIP:        testLoadBalancerIP,
			externalIP:  testExternalIP,
			want:        testExternalIP,
		},
		{
			name:        "lb with lb ip",
			serviceType: vzapi.LoadBalancer,
			lbIP:        testLoadBalancerIP,
			want:        testLoadBalancerIP,
		},
		{
			name:        "lb with external ip",
			serviceType: vzapi.LoadBalancer,
			externalIP:  testExternalIP,
			want:        testExternalIP,
		},
		{
			name:        "nodeport without external ip",
			serviceType: vzapi.NodePort,
			wantErr:     true,
		},
		{
			name:        "nodeport with external ip",
			serviceType: vzapi.NodePort,
			externalIP:  testExternalIP,
			want:        testExternalIP,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vz := &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: tt.serviceType,
						},
					},
				},
			}
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: globalconst.IngressNamespace,
					Name:      vpoconst.NGINXControllerServiceName,
				},
			}
			if len(tt.externalIP) > 0 {
				svc.Spec = corev1.ServiceSpec{
					ExternalIPs: []string{tt.externalIP},
				}
			}
			if len(tt.lbIP) > 0 {
				svc.Status = corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: tt.lbIP},
						},
					},
				}
			}
			fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(svc).Build()
			got, err := GetIngressIP(fakeClient, vz)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIngressIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetIngressIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDNSSuffix(t *testing.T) {
	const testWildCardSuffix = "xip.io"
	testExternalIP := ip.RandomIP()
	testLoadBalancerIP := ip.RandomIP()
	tests := []struct {
		name              string
		serviceType       vzapi.IngressType
		dnsOCIZone        string
		dnsExternalSuffix string
		dnsWildCardSuffix string
		lbIP              string
		externalIP        string
		want              string
		wantErr           bool
	}{
		{
			name:        "lb with oci dns",
			serviceType: vzapi.LoadBalancer,
			dnsOCIZone:  testDomain,
			want:        testDomain,
		},
		{
			name:              "lb with external dns",
			serviceType:       vzapi.LoadBalancer,
			dnsExternalSuffix: testDomain,
			want:              testDomain,
		},
		{
			name:              "lb with external dns and external ip",
			serviceType:       vzapi.LoadBalancer,
			dnsExternalSuffix: testDomain,
			externalIP:        testExternalIP,
			want:              testDomain,
		},
		{
			name:              "lb with wildcard dns and lb ip",
			serviceType:       vzapi.LoadBalancer,
			dnsWildCardSuffix: testWildCardSuffix,
			lbIP:              testLoadBalancerIP,
			want:              testLoadBalancerIP + "." + testWildCardSuffix,
		},
		{
			name:              "lb with wildcard dns and external ip",
			serviceType:       vzapi.LoadBalancer,
			dnsWildCardSuffix: testWildCardSuffix,
			externalIP:        testExternalIP,
			want:              testExternalIP + "." + testWildCardSuffix,
		},
		{
			name:              "lb with wildcard dns and both external and lb ip",
			serviceType:       vzapi.LoadBalancer,
			dnsWildCardSuffix: testWildCardSuffix,
			lbIP:              testLoadBalancerIP,
			externalIP:        testExternalIP,
			want:              testExternalIP + "." + testWildCardSuffix,
		},
		{
			name:        "lb with external ip",
			serviceType: vzapi.LoadBalancer,
			externalIP:  testExternalIP,
			want:        testExternalIP + "." + defaultWildcardDomain,
		},
		{
			name:        "lb with lb ip",
			serviceType: vzapi.LoadBalancer,
			lbIP:        testLoadBalancerIP,
			want:        testLoadBalancerIP + "." + defaultWildcardDomain,
		},
		{
			name:        "lb with both external and lb ip",
			serviceType: vzapi.LoadBalancer,
			lbIP:        testLoadBalancerIP,
			externalIP:  testExternalIP,
			want:        testExternalIP + "." + defaultWildcardDomain,
		},
		{
			name:        "lb no address found",
			serviceType: vzapi.LoadBalancer,
			wantErr:     true,
		},
		{
			name:        "nodeport without external ip",
			serviceType: vzapi.NodePort,
			wantErr:     true,
		},
		{
			name:        "nodeport with external ip",
			serviceType: vzapi.NodePort,
			externalIP:  testExternalIP,
			want:        testExternalIP + "." + defaultWildcardDomain,
		},
		{
			name:              "nodeport with external dns and external ip",
			serviceType:       vzapi.NodePort,
			dnsExternalSuffix: testDomain,
			externalIP:        testExternalIP,
			want:              testDomain,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vz := &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Type: tt.serviceType,
						},
					},
				},
			}
			if len(tt.dnsOCIZone) > 0 {
				vz.Spec.Components.DNS = &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: testDomain,
					},
				}
			} else if len(tt.dnsExternalSuffix) > 0 {
				vz.Spec.Components.DNS = &vzapi.DNSComponent{
					External: &vzapi.External{
						Suffix: tt.dnsExternalSuffix,
					},
				}
			} else if len(tt.dnsWildCardSuffix) > 0 {
				vz.Spec.Components.DNS = &vzapi.DNSComponent{
					Wildcard: &vzapi.Wildcard{
						Domain: tt.dnsWildCardSuffix,
					},
				}
			}
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: globalconst.IngressNamespace,
					Name:      vpoconst.NGINXControllerServiceName,
				},
			}
			if len(tt.externalIP) > 0 {
				svc.Spec = corev1.ServiceSpec{
					ExternalIPs: []string{tt.externalIP},
				}
			}
			if len(tt.lbIP) > 0 {
				svc.Status = corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: tt.lbIP},
						},
					},
				}
			}
			fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(svc).Build()
			got, err := GetDNSSuffix(fakeClient, vz)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDNSSuffix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDNSSuffix() got = %v, want %v", got, tt.want)
			}
		})
	}
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
						DNSZoneName: testDomain,
					},
				},
			},
		},
	}
	domain, err := BuildDNSDomain(fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(), vz)
	assert.NoError(t, err)
	assert.Equal(t, "default."+testDomain, domain)
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
						DNSZoneName: testDomain,
					},
				},
			},
		},
	}
	domain, err := BuildDNSDomain(fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(), vz)
	assert.NoError(t, err)
	assert.Equal(t, "myenv."+testDomain, domain)
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

	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			VolumeClaimSpecTemplates: specTemplateList,
		},
	}
	// Test boundary conditions
	invalidName, found := FindVolumeTemplate("blah", &vz)
	assert.Nil(t, invalidName)
	assert.False(t, found)
	emptyName, found2 := FindVolumeTemplate("", &vz)
	assert.Nil(t, emptyName)
	assert.False(t, found2)
	nilList, found3 := FindVolumeTemplate("default", nil)
	assert.Nil(t, nilList)
	assert.False(t, found3)
	emptyList, found4 := FindVolumeTemplate("default", &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{},
		},
	})
	assert.Nil(t, emptyList)
	assert.False(t, found4)

	// Test normal behavior
	defTemplate, found := FindVolumeTemplate("default", &vz)
	assert.True(t, found)
	assert.Equal(t, "defVolume", defTemplate.VolumeName)
	temp1, found := FindVolumeTemplate("template1", &vz)
	assert.True(t, found)
	assert.Equal(t, "temp1Volume", temp1.VolumeName)
	temp2, found := FindVolumeTemplate("template2", &vz)
	assert.True(t, found)
	assert.Equal(t, "temp2Volume", temp2.VolumeName)
}

// TestGetIngressClassName Tests the GetIngressClassName utility function
// GIVEN a call to GetIngressClassName
// WHEN a Verrazzano resource with an ingress class name specified is given
// THEN the ingress class name specified in the Verrazzano resource is returned
func TestGetIngressClassName(t *testing.T) {
	assert.Equal(t, defaultIngressClassName, GetIngressClassName(&vzapi.Verrazzano{}))
	ingressClassName := "foobar"
	assert.Equal(t, ingressClassName, GetIngressClassName(&vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{
					IngressClassName: &ingressClassName,
				},
			},
		},
	}))
}
