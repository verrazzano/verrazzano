// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package coherence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsCoherenceOperatorReady tests the IsApplicationOperatorReady function
// GIVEN a call to IsCoherenceOperatorReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsCoherenceOperatorReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      coherenceOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	assert.True(t, IsCoherenceOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}

// TestIsCoherenceOperatorNotReady tests the IsApplicationOperatorReady function
// GIVEN a call to IsCoherenceOperatorReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsCoherenceOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      coherenceOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, IsCoherenceOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}
