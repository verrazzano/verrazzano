// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"fmt"
	"testing"

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

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN there is a valid DNS configuration
//  THEN the correct Helm overrides are returned
func TestAppendOverrides(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
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
	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, vz, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
	assert.Equal(t, bom.KeyValue{Key: webFQDNKey, Value: fmt.Sprintf("%s.default.mydomain.com", kialiHostName)}, kvs[1])
}

// TestIsKialiReady tests the isKialiReady function
// GIVEN a call to isKialiReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsKialiReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewFakeClientWithScheme(testScheme,
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
	)

	assert.True(t, isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsKialiNotReady tests the isKialiReady function
// GIVEN a call to isKialiReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
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
	assert.False(t, isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsKialiNotReadyChartNotFound tests the isKialiReady function
// GIVEN a call to isKialiReady
//  WHEN the Kiali chart is not found
//  THEN false is returned
func TestIsKialiNotReadyChartNotFound(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	assert.False(t, isKialiReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}
