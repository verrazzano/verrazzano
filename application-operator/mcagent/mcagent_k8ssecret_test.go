// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateSecretOneMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync Secret objects with a single MultiClusterApplicationConfiguration object
//   containing two secrets
// WHEN the new object exists
// THEN ensure that the Secret objects are created
func TestCreateSecretOneMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	assert.Equal(4, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")
}

// TestCreateSecretTwoMCAppConfigs tests the synchronization method for the following use case.
// GIVEN a request to sync Secret objects with two MultiClusterApplicationConfiguration objects
//   and one of the secrets is shared by two MultiClusterApplicationConfiguration objects
// WHEN the new object exists
// THEN ensure that the Secret objects are created
func TestCreateSecretTwoMCAppConfigs(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	assert.Equal(4, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig,unit-mcappconfig2")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName, "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig", "secret label did not match")
	assert.Contains(secret.Labels[constants2.VerrazzanoManagedLabelKey], constants.LabelVerrazzanoManagedDefault, "secret label did not match")
}

// TestChangePlacement tests the synchronization method for the following use case.
// GIVEN a request to move a MultiClusterApplicationConfiguration object
//   from one cluster to another
// WHEN the new object exists
// THEN ensure that the Secret objects are created and then removed
func TestChangePlacement(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(4, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")

	testMCAppConfig.Spec.Placement.Clusters[0].Name = "managed2"
	err = s.AdminClient.Update(s.Context, &testMCAppConfig)
	assert.NoError(err)

	err = s.syncSecretObjects(testMCAppConfigNamespace)
	assert.NoError(err)

	// Check that secrets have been deleted on the local cluster since the placement has changed
	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))
}

// TestDeleteSecret tests the deletion of secrets for the following use case.
// GIVEN a request to delete a MultiClusterApplicationConfiguration object
//   containing two secrets that are not shared
// WHEN the MultiClusterApplicationConfiguration object is deleted
// THEN ensure that the Secret objects are deleted
func TestDeleteSecret(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(4, len(secret.Labels))
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")
	assertCommonLabels(assert, secret, "unit-mcappconfig")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")

	// Delete the MultiClusterApplicationConfigurarion object from the admin cluster
	err = s.AdminClient.Delete(s.Context, &testMCAppConfig)
	assert.NoError(err)

	// sync
	err = s.syncSecretObjects(testMCAppConfigNamespace)
	assert.NoError(err)

	// Check that secrets have been deleted on the local cluster
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))
}

// TestDeleteSecretSharedSecret tests the deletion of secrets for the following use case.
// GIVEN a request to delete a MultiClusterApplicationConfiguration object
//   containing a secret that is shared by another MultiClusterApplicationConfiguration object
// WHEN the MultiClusterApplicationConfiguration object is deleted
// THEN ensure that the shared secret is not deleted and the verrazzano.io/mc-app-configs label is updated to reflect
//   the deleted MultiClusterApplicationConfiguration object
func TestDeleteSecretSharedSecret(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

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
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(4, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig,unit-mcappconfig2")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(3, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")

	// Delete the MultiClusterApplicationConfigurarion object from the admin cluster
	err = s.AdminClient.Delete(s.Context, &testMCAppConfig1)
	assert.NoError(err)

	// sync
	err = s.syncSecretObjects(testMCAppConfigNamespace)
	assert.NoError(err)

	// shared secret1 should not have been deleted on the local cluster
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)

	// label for secret1 should have been updated
	assert.Contains(secret.Labels[mcAppConfigsLabel], "unit-mcappconfig2", "secret label did not match")

	// secret2 should have been deleted on the local cluster
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))
}

// TestDeleteSecretExtra tests the deletion of secrets for the following use case.
// GIVEN a request to delete a MultiClusterApplicationConfiguration object
//   containing a secret and there is also a secret not part of the MultiClusterApplicationConfiguration object
// WHEN the MultiClusterApplicationConfiguration object is deleted
// THEN ensure that only the secret in the MultiClusterApplicationConfiguration object is deleted
func TestDeleteSecretExtra(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testMCAppConfig2, err := getSampleMCAppConfig("testdata/multicluster-appconfig2.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testSecret1, err := getSampleSecret("testdata/secret1.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	testSecret2, err := getSampleSecret("testdata/secret2.yaml")
	assert.NoError(err, "failed to read sample data for Secret")

	adminClient := fake.NewFakeClientWithScheme(newTestScheme(), &testMCAppConfig2, &testSecret1)

	localClient := fake.NewFakeClientWithScheme(newTestScheme(), &testSecret2)

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}

	err = s.syncSecretObjects(testMCAppConfigNamespace)
	assert.NoError(err)

	// Verify the associated secrets got created on local cluster
	secret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(4, len(secret.Labels))
	assertCommonLabels(assert, secret, "unit-mcappconfig")
	assert.Contains(secret.Labels["label1"], "test1", "secret label did not match")

	secret = &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
	assert.Equal(0, len(secret.Labels))

	// Delete the MultiClusterApplicationConfigurarion object from the admin cluster
	err = s.AdminClient.Delete(s.Context, &testMCAppConfig2)
	assert.NoError(err)

	// sync
	err = s.syncSecretObjects(testMCAppConfigNamespace)
	assert.NoError(err)

	// secret1 should have been deleted on the local cluster
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret1.Name, Namespace: testSecret1.Namespace}, secret)
	assert.True(apierrors.IsNotFound(err))
	// secret2 should not have been deleted on the local cluster
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testSecret2.Name, Namespace: testSecret2.Namespace}, secret)
	assert.NoError(err)
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

// assert labels common to all K8S secrets synced to managed cluster
func assertCommonLabels(assert *asserts.Assertions, secret *corev1.Secret, appConfigs string) {
	assert.Contains(secret.Labels[managedClusterLabel], testClusterName, "secret label did not match")
	assert.Contains(secret.Labels[mcAppConfigsLabel], appConfigs, "secret label did not match")
	assert.Contains(secret.Labels[constants2.VerrazzanoManagedLabelKey], constants.LabelVerrazzanoManagedDefault, "secret label did not match")
}

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	return scheme
}
