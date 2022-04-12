// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Ingress: &vzapi.IngressNginxComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// TestAppendNGINXOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass a VZ spec with defaults
//  THEN the values created properly
func TestAppendNGINXOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
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
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
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
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
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
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, false), ComponentName, ComponentNamespace, "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 2)
}

// TestNGINXPreInstall tests the PreInstall fn
// GIVEN a call to this fn
//  WHEN I call PreInstall
//  THEN no errors are returned
func TestNGINXPreInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	err := PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, false), ComponentName, ComponentNamespace, "")
	assert.NoError(t, err)
}

// TestIsNGINXReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsNGINXReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ControllerName,
			Labels:    map[string]string{"app.kubernetes.io/component": "controller"},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      backendName,
				Labels:    map[string]string{"app.kubernetes.io/component": "default-backend"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	assert.True(t, isNginxReady(spi.NewFakeContext(fakeClient, nil, false)))
}

// TestIsNGINXNotReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsNGINXNotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ControllerName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      backendName,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	assert.False(t, isNginxReady(spi.NewFakeContext(fakeClient, nil, false)))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(svc).Build()
	err := PostInstall(spi.NewFakeContext(fakeClient, vz, false), ComponentName, ComponentNamespace)
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	assert.NoError(t, PostInstall(spi.NewFakeContext(fakeClient, vz, false), ComponentName, ComponentNamespace))
}

// TestPostInstallDryRun tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the context DryRun flag is true
//  THEN no error is returned
func TestPostInstallDryRun(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	assert.NoError(t, PostInstall(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false), ComponentName, ComponentNamespace))
}

// TestNewComponent tests the NewComponent function
// GIVEN a call to NewComponent
//  THEN the NGINX component is returned
func TestNewComponent(t *testing.T) {
	component := NewComponent()
	assert.NotNil(t, component)
	assert.Equal(t, ComponentName, component.Name())
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Nginx component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(&vzapi.Verrazzano{}))
}

// TestIsEnabledNilNginx tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Nginx component is nil
//  THEN true is returned
func TestIsEnabledNilNginx(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Ingress = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Nginx component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Ingress.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Nginx component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Ingress.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Nginx component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Ingress.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(&cr))
}

func getBoolPtr(b bool) *bool {
	return &b
}
