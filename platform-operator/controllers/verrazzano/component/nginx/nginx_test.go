// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package nginx

import (
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
			Name:      controllerName,
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
				Name:      backendName,
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
			Name:      controllerName,
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
				Name:      backendName,
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
		ObjectMeta: metav1.ObjectMeta{Name: controllerName, Namespace: ComponentNamespace},
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

// Test_getServiceTypeLoadBalancer tests the getServiceType function
// GIVEN a call to getServiceType
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

	svcType, err := getServiceType(vz)
	assert.NoError(t, err)
	assert.Equal(t, vzapi.LoadBalancer, svcType)
}

// Test_getServiceTypeNodePort tests the getServiceType function
// GIVEN a call to getServiceType
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

	svcType, err := getServiceType(vz)
	assert.NoError(t, err)
	assert.Equal(t, vzapi.NodePort, svcType)
}

// Test_getServiceTypeInvalidType tests the getServiceType function
// GIVEN a call to getServiceType
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

	svcType, err := getServiceType(vz)
	assert.Error(t, err)
	assert.Equal(t, vzapi.IngressType(""), svcType)
}
