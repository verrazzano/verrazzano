// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestIsReady tests the AuthProxy IsReady call
// GIVEN a AuthProxy component
//  WHEN I call IsReady when all requirements are met
//  THEN true or false is returned
func TestIsReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			name: "Test IsReady when AuthProxy is successfully deployed",
			client: fake.NewFakeClientWithScheme(k8scheme.Scheme,
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 0,
					},
				}),
			expectTrue: true,
		},
		{
			name: "Test IsReady when AuthProxy deployment is not ready",
			client: fake.NewFakeClientWithScheme(k8scheme.Scheme,
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						ReadyReplicas:       1,
						AvailableReplicas:   0,
						UnavailableReplicas: 1,
					},
				}),
			expectTrue: false,
		},
		{
			name:       "Test IsReady when AuthProxy deployment does not exist",
			client:     fake.NewFakeClientWithScheme(k8scheme.Scheme),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			if tt.expectTrue {
				assert.True(t, NewComponent().IsReady(ctx))
			} else {
				assert.False(t, NewComponent().IsReady(ctx))
			}
		})
	}
}

// TestIsEnabled tests the AuthProxy IsEnabled call
// GIVEN a AuthProxy component
//  WHEN I call IsEnabled when all requirements are met
//  THEN true or false is returned
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			name: "Test IsEnabled when using AuthProxy component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, false, profilesRelativePath)
			if tt.expectTrue {
				assert.True(t, NewComponent().IsEnabled(ctx))
			} else {
				assert.False(t, NewComponent().IsEnabled(ctx))
			}
		})
	}
}

// TestGetIngressNames tests the AuthProxy GetIngressNames call
// GIVEN a AuthProxy component
//  WHEN I call GetIngressNames
//  THEN the correct list of names is returned
func TestGetIngressNames(t *testing.T) {
	ingressNames := NewComponent().GetIngressNames(nil)
	assert.True(t, len(ingressNames) == 1)
	assert.Equal(t, constants.VzConsoleIngress, ingressNames[0].Name)
	assert.Equal(t, ComponentNamespace, ingressNames[0].Namespace)
}
