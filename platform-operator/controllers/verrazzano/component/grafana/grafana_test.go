// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()
var replicas int32

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = vmov1.AddToScheme(testScheme)
}

// TestIsGrafanaInstalled tests the isGrafanaInstalled function for the Grafana component
func TestIsGrafanaInstalled(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Grafana deployment exists
			// WHEN we call isGrafanaInstalled
			// THEN the call returns true
			name: "Test isGrafanaInstalled when Grafana is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Grafana deployment does not exist
			// WHEN we call isGrafanaInstalled
			// THEN the call returns false
			name:       "Test isGrafanaInstalled when Grafana deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isGrafanaInstalled(ctx))
		})
	}
}

// TestIsGrafanaReady tests the isGrafanaReady function
func TestIsGrafanaReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Grafana deployment exists and there are available replicas
			// AND the Grafana admin secret exists
			// WHEN we call isGrafanaReady
			// THEN the call returns true
			name: "Test isGrafanaReady when Grafana is successfully deployed and the admin secret exists",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
						Labels:    map[string]string{"app": "system-grafana"},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "system-grafana"},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"app":               "system-grafana",
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        grafanaDeployment + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.GrafanaSecret,
						Namespace: ComponentNamespace,
					},
					Data: map[string][]byte{},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Grafana deployment exists and there are available replicas
			// AND the Grafana admin secret does not exist
			// WHEN we call isGrafanaReady
			// THEN the call returns false
			name: "Test isGrafanaReady when Grafana is successfully deployed and the admin secret does not exist",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Grafana deployment exists and there are no available replicas
			// WHEN we call isGrafanaReady
			// THEN the call returns false
			name: "Test isGrafanaReady when Grafana is deployed but there are no available replicas",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isGrafanaReady(ctx))
			ctx = spi.NewFakeContext(tt.client, &vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Grafana: &vzapi.GrafanaComponent{Replicas: &replicas}}}}, nil, false)
			assert.Equal(t, true, isGrafanaReady(ctx))
		})
	}
}
