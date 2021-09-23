// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"os"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestAppendAppOperatorOverrides tests the Keycloak override for the theme images
// GIVEN an env override for the app operator image
//  WHEN I call AppendApplicationOperatorOverrides
//  THEN the "image" Key is set with the image override.
func TestAppendAppOperatorOverrides(t *testing.T) {
	assert := assert.New(t)

	customImage := "myreg.io/myrepo/v8o/verrazzano-application-operator-dev:local-20210707002801-b7449154"

	kvs, err := AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 0, "AppendApplicationOperatorOverrides returned an unexpected number of Key:Value pairs")

	os.Setenv(constants.VerrazzanoAppOperatorImageEnvVar, customImage)
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err = AppendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "AppendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 1, "AppendApplicationOperatorOverrides returned wrong number of Key:Value pairs")
	assert.Equalf("image", kvs[0].Key, "Did not get expected image Key")
	assert.Equalf(customImage, kvs[0].Value, "Did not get expected image Value")
}

// TestIsApplicationOperatorReady tests the IsApplicationOperatorReady function
// GIVEN a call to IsApplicationOperatorReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsApplicationOperatorReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "verrazzano-application-operator",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	assert.True(t, IsApplicationOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}

// TestIsApplicationOperatorNotReady tests the IsApplicationOperatorReady function
// GIVEN a call to IsApplicationOperatorReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsApplicationOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "verrazzano-application-operator",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, IsApplicationOperatorReady(zap.S(), fakeClient, "", constants.VerrazzanoSystemNamespace))
}
