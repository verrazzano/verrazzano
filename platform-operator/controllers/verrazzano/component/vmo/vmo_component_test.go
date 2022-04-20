// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestIsEnabled tests the VMO IsEnabled call
// GIVEN a VMO component
//  WHEN I call IsEnabled
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
			name: "Test IsEnabled when all VMI component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							MonitoringComponent: vzapi.MonitoringComponent{
								Enabled: &falseValue,
							},
						},
						Kibana: &vzapi.KibanaComponent{
							MonitoringComponent: vzapi.MonitoringComponent{
								Enabled: &falseValue,
							},
						},
						Grafana: &vzapi.GrafanaComponent{
							MonitoringComponent: vzapi.MonitoringComponent{
								Enabled: &falseValue,
							},
						},
						Prometheus: &vzapi.PrometheusComponent{
							MonitoringComponent: vzapi.MonitoringComponent{
								Enabled: &falseValue,
							},
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
				assert.True(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			} else {
				assert.False(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			}
		})
	}
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsReady(t *testing.T) {
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
	assert.True(t, NewComponent().IsReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, true)))
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the VMO is not ready per Helm
//  THEN true is returned
func TestIsNotReady(t *testing.T) {
	assert.False(t, NewComponent().IsReady(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false)))
}

// TestPreInstall tests the VMO PreInstall call
// GIVEN a VMO component
//  WHEN I call PreInstall with defaults
//  THEN no error is returned
func TestPreInstall(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	scheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	err := NewComponent().PreInstall(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(scheme).Build(), nil, false))
	assert.NoError(t, err)
}

// TestPreUpgrade tests the VMO PreUpgrade call
// GIVEN a VMO component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../helm_config"
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), nil, false))
	assert.NoError(t, err)
}
