// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"testing"

	"github.com/stretchr/testify/assert"
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
	profileDir      = "../../../../../manifests/profiles"
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

// TestIsJaegerOperatorReady tests the isJaegerOperatorReady function for the Jaeger Operator
func TestIsJaegerOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Jaeger Operator deployment exists and there are available replicas
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Jaeger Operator is successfully deployed",
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
			// GIVEN the Jaeger Operator deployment exists and there are no available replicas
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Jaeger Operator deployment is not ready",
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
			// GIVEN the Jaeger Operator deployment does not exist
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isJaegerOperatorReady(ctx))
		})
	}
}

// TestPreInstall tests the preInstall function.
func TestPreInstall(t *testing.T) {
	// GIVEN the Jaeger Operator is being installed
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

// TestValidateJaegerOperator tests the validation of the Jaeger Operator installation and the Verrazzano CR
func TestValidateJaegerOperator(t *testing.T) {
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		expectError bool
	}{
		{
			name:        "test nothing enabled",
			vz:          vzapi.Verrazzano{},
			expectError: false,
		},
		{
			name: "test jaeger operator enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test jaeger operator disabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: false,
		},
	}
	c := jaegerOperatorComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validateJaegerOperator(&tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
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

	// GIVEN a Verrazzano CR
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values for Jaeger images
	vz := &vzapi.Verrazzano{}

	ctx := spi.NewFakeContext(client, vz, false)

	var err error
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)

	assert.Equal(t, "ghcr.io/verrazzano/jaeger-agent:1.32.0", bom.FindKV(kvs, "jaegerAgentImage"))
	assert.Equal(t, "ghcr.io/verrazzano/jaeger-query:1.32.0", bom.FindKV(kvs, "jaegerQueryImage"))
	assert.Equal(t, "ghcr.io/verrazzano/jaeger-collector:1.32.0", bom.FindKV(kvs, "jaegerCollectorImage"))
	assert.Equal(t, "ghcr.io/verrazzano/jaeger-ingester:1.32.0", bom.FindKV(kvs, "jaegerIngesterImage"))
}

// TestEnsureMonitoringOperatorNamespace asserts the verrazzano-monitoring namespaces can be created
func TestEnsureMonitoringOperatorNamespace(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, false)
	err := ensureVerrazzanoMonitoringNamespace(ctx)
	assert.NoError(t, err)
}
