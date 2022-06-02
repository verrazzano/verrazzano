// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../../testdata/test_bom.json"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsPrometheusOperatorReady tests the isPrometheusOperatorReady function for the Prometheus Operator
func TestIsPrometheusOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Prometheus Operator deployment exists and there are available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Prometheus Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				}).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Operator deployment exists and there are no available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Prometheus Operator deployment does not exist
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Prometheus Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isPrometheusOperatorReady(ctx))
		})
	}
}

// TestAppendOverrides tests that helm overrides are set properly
func TestAppendOverrides(t *testing.T) {
	oldBomPath := config.GetDefaultBOMFilePath()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath(oldBomPath)

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	kvs := make([]bom.KeyValue, 0)

	// GIVEN a Verrazzano CR with the CertManager component enabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, vz, false)

	var err error
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 7)

	assert.Equal(t, "ghcr.io/verrazzano/prometheus-config-reloader", bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.tag"))

	assert.Equal(t, "ghcr.io/verrazzano/alertmanager", bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.tag"))

	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.alertmanagerDefaultBaseImage"), "ghcr.io/verrazzano/alertmanager:"))
	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.prometheusDefaultBaseImage"), "ghcr.io/verrazzano/prometheus:"))

	assert.Equal(t, "true", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))

	// GIVEN a Verrazzano CR with the CertManager component disabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to false
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	ctx = spi.NewFakeContext(client, vz, false)
	kvs = make([]bom.KeyValue, 0)

	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 7)

	assert.Equal(t, "false", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))
}

// TestPreInstall tests the preInstall function.
func TestPreInstall(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed
	// WHEN the preInstall function is called
	// THEN the component namespace is created in the cluster
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := preInstall(ctx)
	assert.NoError(t, err)

	ns := v1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestGetHelmOverridesPromNotConfigured tests the preInstall function.
// GIVEN a call to GetHelmOverrides
// WHEN the PrometheusOperator component is nil (not configured)
// THEN an empty override set is returned
func TestGetHelmOverridesPromNotConfigured(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	// Nil Prom Operator component
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	overrides := GetOverrides(ctx.EffectiveCR())
	assert.Equal(t, []vzapi.Overrides{}, overrides)
}

// TestGetHelmOverrides tests the preInstall function.
// GIVEN a call to GetHelmOverrides
// WHEN the PrometheusOperator component has overrides configured
// THEN the correct override set is returned
func TestGetHelmOverrides(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	expectedOverrides := []vzapi.Overrides{
		{ConfigMapRef: &v1.ConfigMapKeySelector{
			Key: "foo",
		}},
	}
	ctx := spi.NewFakeContext(client,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					PrometheusOperator: &vzapi.PrometheusOperatorComponent{
						InstallOverrides: vzapi.InstallOverrides{
							ValueOverrides: expectedOverrides,
						},
					},
				},
			},
		},
		false,
	)

	overrides := GetOverrides(ctx.EffectiveCR())
	assert.Equal(t, expectedOverrides, overrides)
}
