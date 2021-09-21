// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package weblogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test_appendWeblogicOperatorOverridesExtraKVs tests the AppendWeblogicOperatorOverrides fn
// GIVEN a call to AppendWeblogicOperatorOverrides
//  WHEN I call with no extra kvs
//  THEN the correct number of KeyValue objects are returned and no errors occur
func Test_appendWeblogicOperatorOverrides(t *testing.T) {
	kvs, err := AppendWeblogicOperatorOverrides(zap.S(), "weblogic-operator", "verrazzano-system", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
}

// Test_appendWeblogicOperatorOverridesExtraKVs tests the AppendWeblogicOperatorOverrides fn
// GIVEN a call to AppendWeblogicOperatorOverrides
//  WHEN I pass in a KeyValue list
//  THEN the values passed in are preserved and no errors occur
func Test_appendWeblogicOperatorOverridesExtraKVs(t *testing.T) {
	kvs := []bom.KeyValue{
		{Key: "Key", Value: "Value"},
	}
	var err error
	kvs, err = AppendWeblogicOperatorOverrides(zap.S(), "weblogic-operator", "verrazzano-system", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 5)
}

// Test_weblogicOperatorPreInstall tests the WeblogicOperatorPreInstall fn
// GIVEN a call to this fn
//  WHEN I call WeblogicOperatorPreInstall
//  THEN no errors are returned
func Test_weblogicOperatorPreInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	kvs, err := WeblogicOperatorPreInstall(zap.S(), client, "weblogic-operator", "verrazzano-system", "")
	assert.NoError(t, err)
	assert.Len(t, kvs, 0)
}

// TestIsWeblogicOperatorReady tests the IsWeblogicOperatorReady function
// GIVEN a call to IsWeblogicOperatorReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsWeblogicOperatorReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      wlsOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	assert.True(t, IsWeblogicOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}

// TestIsWeblogicOperatorNotReady tests the IsWeblogicOperatorReady function
// GIVEN a call to IsWeblogicOperatorReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsWeblogicOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      wlsOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, IsWeblogicOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}
