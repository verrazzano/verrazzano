// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath         = "../../../testdata/test_bom.json"
)

const extraEnvValue= `
  - name: "JAEGER-AGENT-IMAGE"
    value: "ghcr.io/verrazzano/jaeger-agent:1.32.0"
  - name: "JAEGER-QUERY-IMAGE"
    value: "ghcr.io/verrazzano/jaeger-query:1.32.0"
  - name: "JAEGER-COLLECTOR-IMAGE"
    value: "ghcr.io/verrazzano/jaeger-collector:1.32.0"
  - name: "JAEGER-INGESTER-IMAGE"
    value: "ghcr.io/verrazzano/jaeger-ingester:1.32.0"
`

var testScheme = runtime.NewScheme()

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

// TestAppendOverrides tests that the Jaeger Operator overrides are generated correctly.
// GIVEN a Verrazzano BOM
// WHEN I call AppendOverrides
// THEN the Jaeger Operator overrides Key:Value array has the expected content.
func TestAppendOverrides(t *testing.T) {
	a := assert.New(t)

	const env = "test-env"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: env,
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendOverrides(spi.NewFakeContext(c, vz, false), "", "", "", nil)

	a.NoError(err, "AppendOverrides returned an error")
	a.Len(kvs, 1, "AppendOverrides returned wrong number of Key:Value pairs")

	a.Contains(kvs, bom.KeyValue{
		Key:       extraEnvKey,
		Value:     extraEnvValue,
		SetString: true,
	})
}
