// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"context"
	"fmt"
	"testing"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type erroringFakeClient struct {
	client.Client
}

func (e erroringFakeClient) Get(_ context.Context, _ types.NamespacedName, _ client.Object) error {
	return fmt.Errorf("server-error")
}

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//
//	WHEN there is a valid DNS configuration
//	THEN the correct Helm overrides are returned, else return an error
func TestAppendOverrides(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kialiSigningKeySecret,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	erroringClient := erroringFakeClient{fakeClient}
	fakeClientWithObject := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&secret).Build()
	validVZ := &vzapi.Verrazzano{
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
	invalidVZ := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "",
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		vz          *vzapi.Verrazzano
		client      client.Client
		wantErr     bool
		expectedLen int
	}{
		{
			"Valid DNS Configuration, No error",
			validVZ,
			fakeClient,
			false,
			3,
		},
		{
			"Invalid DNS Configuration, error",
			invalidVZ,
			fakeClient,
			true,
			1,
		},
		{
			"Valid DNS Configuration, Server error",
			validVZ,
			erroringClient,
			true,
			2,
		},
		{
			"Valid DNS Configuration, Error retrieving key",
			validVZ,
			fakeClientWithObject,
			true,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs, err := AppendOverrides(spi.NewFakeContext(tt.client, tt.vz, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
			assert.Equal(t, len(kvs), tt.expectedLen)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
				assert.Equal(t, bom.KeyValue{Key: webFQDNKey, Value: fmt.Sprintf("%s.default.mydomain.com", kialiHostName)}, kvs[1])
				assert.Equal(t, signingKeyPath, kvs[2].Key)
			}
		})
	}
}

// TestIsKialiReady tests the isKialiReady function
// GIVEN a call to isKialiReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsKialiReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      kialiSystemName,
				Labels:    map[string]string{"app": kialiSystemName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": kialiSystemName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      kialiSystemName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               kialiSystemName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        kialiSystemName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()

	kiali := NewComponent().(kialiComponent)
	assert.True(t, kiali.isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsKialiNotReady tests the isKialiReady function
// GIVEN a call to isKialiReady
//
//	WHEN the deployment object does NOT have enough replicas available
//	THEN false is returned
func TestIsKialiNotReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	kiali := NewComponent().(kialiComponent)
	assert.False(t, kiali.isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsKialiNotReadyChartNotFound tests the isKialiReady function
// GIVEN a call to isKialiReady
//
//	WHEN the Kiali chart is not found
//	THEN false is returned
func TestIsKialiNotReadyChartNotFound(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	kiali := NewComponent().(kialiComponent)
	assert.False(t, kiali.isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

func TestGetOverrides(t *testing.T) {
	overridesValue := []byte("{\"key1\": \"value1\"}")
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kiali: &vzapi.KialiComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: overridesValue},
							},
						},
					},
				},
			},
		},
	}

	comp := NewComponent()
	overrides := comp.GetOverrides(vz).([]vzapi.Overrides)
	assert.Equal(t, overridesValue, overrides[0].Values.Raw)

	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Kiali: nil,
			},
		},
	}

	overrides = comp.GetOverrides(vz).([]vzapi.Overrides)
	assert.Equal(t, []vzapi.Overrides{}, overrides)

	beta1vz := &installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Kiali: &installv1beta1.KialiComponent{
					InstallOverrides: installv1beta1.InstallOverrides{
						ValueOverrides: []installv1beta1.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: overridesValue},
							},
						},
					},
				},
			},
		},
	}

	overrides2 := comp.GetOverrides(beta1vz).([]installv1beta1.Overrides)
	assert.Equal(t, overridesValue, overrides2[0].Values.Raw)
}
