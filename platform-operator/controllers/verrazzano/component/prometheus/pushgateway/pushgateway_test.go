// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsPushgatewayReady tests the isPushgatewayReady function for the PrometheusPushgateway
func TestIsPushgatewayReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the PrometheusPushgateway deployment exists and there are available replicas
			// WHEN we call isPushgatewayReady
			// THEN the call returns true
			name: "Test IsReady when PrometheusPushgateway is successfully deployed",
			client: fake.NewFakeClientWithScheme(testScheme,
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
						Labels:    map[string]string{"app": ComponentName},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": ComponentName},
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
						Name:      deploymentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"app":               ComponentName,
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        deploymentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			),
			expectTrue: true,
		},
		{
			// GIVEN the PrometheusPushgateway deployment exists and there are no available replicas
			// WHEN we call isPushgatewayReady
			// THEN the call returns false
			name: "Test IsReady when PrometheusPushgateway deployment is not ready",
			client: fake.NewFakeClientWithScheme(testScheme,
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
				}),
			expectTrue: false,
		},
		{
			// GIVEN the PrometheusPushgateway deployment does not exist
			// WHEN we call isPushgatewayReady
			// THEN the call returns false
			name:       "Test IsReady when PrometheusPushgateway deployment does not exist",
			client:     fake.NewFakeClientWithScheme(testScheme),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isPushgatewayReady(ctx))
		})
	}
}
