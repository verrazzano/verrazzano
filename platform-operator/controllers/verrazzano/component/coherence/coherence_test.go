// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package coherence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsCoherenceOperatorReady tests the isCoherenceOperatorReady function
// GIVEN a call to isCoherenceOperatorReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsCoherenceOperatorReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
			Labels:    map[string]string{"control-plane": "coherence"},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	})
	assert.True(t, isCoherenceOperatorReady(spi.NewFakeContext(fakeClient, nil, false)))
}

// TestIsCoherenceOperatorNotReady tests the isCoherenceOperatorReady function
// GIVEN a call to isCoherenceOperatorReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsCoherenceOperatorNotReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	})
	assert.False(t, isCoherenceOperatorReady(spi.NewFakeContext(fakeClient, nil, false)))
}
