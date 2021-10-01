// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateSecretOneAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync Secret objects with a single MultiClusterApplicationConfiguration object
//   containing two secrets
// WHEN the new object exists
// THEN ensure that the Secret objects are created
func TestCreateSecretOneAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testSecret1, err := getSampleSecret("testdata/secret1.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	testSecret2, err := getSampleSecret("testdata/secret2.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	adminClient := fake.NewFakeClientWithScheme(newTestScheme(), &testMCAppConfig, &testSecret1, &testSecret2)

	localClient := fake.NewFakeClientWithScheme(newTestScheme())

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}

	err = s.syncSecretObjects(testMCAppConfigNamespace)

	// Validate the results
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName,  "secret label did not match")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(2, len(secret.Labels))
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName,  "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig", "secret label did not match")
}

// TestCreateSecretTwoAppConfigs tests the synchronization method for the following use case.
// GIVEN a request to sync Secret objects with a two MultiClusterApplicationConfiguration object using
//   containing two secrets - one the secrets is shared by two MultiClusterApplicationConfiguration objects
// WHEN the new object exists
// THEN ensure that the Secret objects are created
func TestCreateSecretTwoAppConfigs(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Test data
	testMCAppConfig1, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testMCAppConfig2, err := getSampleMCAppConfig("testdata/multicluster-appconfig2.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testSecret1, err := getSampleSecret("testdata/secret1.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	testSecret2, err := getSampleSecret("testdata/secret2.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	adminClient := fake.NewFakeClientWithScheme(newTestScheme(), &testMCAppConfig1, &testMCAppConfig2, &testSecret1, &testSecret2)

	localClient := fake.NewFakeClientWithScheme(newTestScheme())

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}

	err = s.syncSecretObjects(testMCAppConfigNamespace)

	// Validate the results
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName,  "secret label did not match")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig,unit-mcappconfig2", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(2, len(secret.Labels))
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName,  "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig", "secret label did not match")
}

// getSampleSecret returns a sample secret object
func getSampleSecret(filePath string) (corev1.Secret, error) {
	secret := corev1.Secret{}
	sampleSecretFile, err := filepath.Abs(filePath)
	if err != nil {
		return secret, err
	}

	rawResource, err := clusterstest.ReadYaml2Json(sampleSecretFile)
	if err != nil {
		return secret, err
	}

	err = json.Unmarshal(rawResource, &secret)
	return secret, err
}

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	return scheme
}
