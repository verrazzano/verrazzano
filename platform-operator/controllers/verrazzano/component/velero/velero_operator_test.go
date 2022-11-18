// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"reflect"
	"testing"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

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
	_ = appsv1.AddToScheme(testScheme)
}

// TestisVeleroOperatorReady tests the isVeleroOperatorReady function for the Velero Operator
func TestIsVeleroOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Velero Operator deployment exists and there are available replicas
			// WHEN we call isVeleroOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Velero Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
						Labels:    map[string]string{"name": deploymentName},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"name": deploymentName},
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
							"name":              deploymentName,
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

				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      constants.ResticDaemonSetName,
						Labels:    map[string]string{"name": constants.ResticDaemonSetName},
					},
					Spec: appsv1.DaemonSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"name": constants.ResticDaemonSetName},
						},
					},
					Status: appsv1.DaemonSetStatus{
						UpdatedNumberScheduled: 1,
						NumberAvailable:        1,
						NumberMisscheduled:     1,
						NumberReady:            1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      constants.ResticDaemonSetName,
						Labels: map[string]string{
							"name":                     constants.ResticDaemonSetName,
							"controller-revision-hash": "restic-95d8c5d96",
						},
					},
				},
				&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "restic-restic-95d8c5d96",
						Namespace: ComponentNamespace,
					},
					Revision: 1,
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Velero Operator deployment exists and there are no available replicas
			// WHEN we call isVeleroOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Velero Operator deployment is not ready",
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
			// GIVEN the Velero Operator deployment does not exist
			// WHEN we call isVeleroOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Velero Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isVeleroOperatorReady(ctx))
		})
	}
}

// TestGetOverrides tests GetOverrides to fetch all the Velero Overrides
func TestGetOverrides(t *testing.T) {
	tests := []struct {
		name   string
		object runtime.Object
		want   interface{}
	}{
		// GIVEN nil
		// WHEN GetOverrides is called
		// THEN empty list of Overrides are returned
		{
			"TestGetOverridesWithInvalidOverrides",
			nil,
			[]vzapi.Overrides{},
		},
		// GIVEN v1alpha1 VZ CR with no Velero component
		// WHEN GetOverrides is called
		// THEN empty list of Overrides is returned
		{
			"TestGetOverridesWithNoVelero",
			&vzapi.Verrazzano{},
			[]vzapi.Overrides{},
		},
		// GIVEN v1beta1 VZ CR with no Velero component
		// WHEN GetOverrides is called
		// THEN empty list of Overrides is returned
		{
			"TestGetOverridesWithV1Beta CR with no Velero",
			&installv1beta1.Verrazzano{},
			[]installv1beta1.Overrides{},
		},
		// GIVEN VZ CR with Velero Overrides
		// WHEN GetOverrides is called
		// THEN list of Velero Overrides is returned
		{
			"TestGetOverridesWithVeleroOverrides",
			getSingleOverrideCR(),
			[]vzapi.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(validOverrideJSON),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOverrides(tt.object); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}
