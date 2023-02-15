// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package coherence

import (
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsCoherenceOperatorReady tests the isCoherenceOperatorReady function
// GIVEN a call to isCoherenceOperatorReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsCoherenceOperatorReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"control-plane": "coherence"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"control-plane": "coherence"},
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
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"control-plane":     "coherence",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	coh := coherenceComponent{
		HelmComponent: helm.HelmComponent{
			ChartDir:               config.GetThirdPartyDir(),
			ImagePullSecretKeyname: secret.DefaultImagePullSecretKeyName,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
	assert.True(t, coh.isCoherenceOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}

// TestIsCoherenceOperatorNotReady tests the isCoherenceOperatorReady function
// GIVEN a call to isCoherenceOperatorReady
//
//	WHEN the deployment object does NOT have enough replicas available
//	THEN false is returned
func TestIsCoherenceOperatorNotReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	coh := coherenceComponent{
		HelmComponent: helm.HelmComponent{
			ChartDir:               config.GetThirdPartyDir(),
			ImagePullSecretKeyname: secret.DefaultImagePullSecretKeyName,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
	assert.False(t, coh.isCoherenceOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}

func TestGetOverrides(t *testing.T) {
	ref := &corev1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := vzapi.InstallOverrides{
		ValueOverrides: []vzapi.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	var tests = []struct {
		name string
		cr   runtime.Object
		res  interface{}
	}{
		{
			"overrides when component not nil, v1alpha1",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CoherenceOperator: &vzapi.CoherenceOperatorComponent{
							InstallOverrides: oV1Alpha1,
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"Empty overrides when component nil",
			&v1beta1.Verrazzano{},
			[]v1beta1.Overrides{},
		},
		{
			"overrides when component not nil",
			&v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CoherenceOperator: &v1beta1.CoherenceOperatorComponent{
							InstallOverrides: o,
						},
					},
				},
			},
			o.ValueOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}
