// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomPath = "../../../../../verrazzano-bom.json"

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = appsv1.AddToScheme(testScheme)
}

//TestBuildInstallArgs verifies the install args are present as expected from the BOM
func TestBuildInstallArgs(t *testing.T) {
	defer config.Set(config.Get())

	var tests = []struct {
		name     string
		bomFile  string
		hasError bool
	}{
		{
			"build install args from valid bom",
			testBomPath,
			false,
		},
		{
			"fails to build install args when bomfile doesn't exist",
			"invalid bom file",
			true,
		},
		{
			"fails to build install args when bomfile doesn't have jaeger subcomponent",
			"invalid bom file",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetDefaultBomFilePath(tt.bomFile)
			args, err := buildInstallArgs()
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for _, subcomponent := range subcomponentNames {
					img, ok := args[strings.ReplaceAll(subcomponent, "-", "")]
					assert.True(t, ok, fmt.Sprintf("couldn't find subcomponent image: %s", img))
					assert.Contains(t, img, subcomponent)
				}
			}
		})
	}
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

// TestEnsureMonitoringOperatorNamespace asserts the verrazzano-monitoring namespaces can be created
func TestEnsureMonitoringOperatorNamespace(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, false)
	err := ensureVerrazzanoMonitoringNamespace(ctx)
	assert.NoError(t, err)
}
