// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestIsVmoReady tests the isVmoReady function
// GIVEN a call to isVmoReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsVmoReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
			Labels:    map[string]string{"k8s-app": ComponentName},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	assert.True(t, isVmoReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsVmoNotReady tests the isVmoReady function
// GIVEN a call to isVmoReady
//  WHEN the deployment object does not have enough replicas available
//  THEN true is returned
func TestIsVmoNotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
			Labels:    map[string]string{"k8s-app": ComponentName},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 0,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	assert.False(t, isVmoReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestAppendVmoOverrides tests the appendVmoOverrides function
// GIVEN a call to appendVmoOverrides
//  WHEN I call with no extra kvs
//  THEN the correct KeyValue objects are returned and no error occurs
func TestAppendVmoOverrides(t *testing.T) {
	a := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ingress-nginx",
			Name:      "ingress-controller-ingress-nginx-controller",
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{
				"nn.nn.nn.nn",
			},
		},
	}).Build()

	kvs, err := appendVmoOverrides(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false), "", "", "", []bom.KeyValue{})

	a.NoError(err)
	a.Len(kvs, 4)
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.prometheusInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7-slim",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.esInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7.8",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "config.dnsSuffix",
		Value: "nn.nn.nn.nn.nip.io",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "config.envName",
		Value: "default",
	})
}

// TestAppendVMOOverridesNoNGINX tests the appendVmoOverrides function
// GIVEN a call to appendVmoOverrides
//  WHEN I call with no extra kvs and NGINX is disabled
//  THEN the correct KeyValue objects are returned and no error occurs
func TestAppendVmoOverridesNoNGINX(t *testing.T) {
	a := assert.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	enabled := false
	kvs, err := appendVmoOverrides(spi.NewFakeContext(fakeClient,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Ingress: &vzapi.IngressNginxComponent{
						Enabled: &enabled,
					},
				},
			},
		},
		false), "", "", "", []bom.KeyValue{})

	a.NoError(err)
	a.Len(kvs, 2)
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.prometheusInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7-slim",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   "monitoringOperator.esInitImage",
		Value: "ghcr.io/oracle/oraclelinux:7.8",
	})
}
