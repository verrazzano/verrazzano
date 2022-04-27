// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSyncAdminCANoDifference tests the synchronization method for the following use case.
// GIVEN a request to sync Admin CA certs
// WHEN the CAs are the same
// THEN ensure that no secret is updated.
func TestSyncCACertsNoDifference(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testAdminCASecret, err := getSampleClusterCASecret("testdata/clusterca-admincasecret.yaml")
	assert.NoError(err, "failed to read sample Admin CA Secret")

	testClusterRegSecret, err := getSampleClusterCASecret("testdata/clusterca-clusterregsecret.yaml")
	assert.NoError(err, "failed to read sample Cluster Registration Secret")

	testMCTLSSecret, err := getSampleClusterCASecret("testdata/clusterca-mctlssecret.yaml")
	assert.NoError(err, "failed to read sample MC TLS Secret")

	testMCCASecret, err := getSampleClusterCASecret("testdata/clusterca-mccasecret.yaml")
	assert.NoError(err, "failed to read sample MC CA Secret")

	testVMC, err := getSampleClusterCAVMC("testdata/clusterca-vmc.yaml")
	assert.NoError(err, "failed to read sample VMC")

	origRegCA := testClusterRegSecret.Data["ca-bundle"]
	origMCCA := testMCCASecret.Data["cacrt"]

	adminClient := fake.NewFakeClientWithScheme(newClusterCAScheme(), &testAdminCASecret, &testMCCASecret, &testVMC)

	localClient := fake.NewFakeClientWithScheme(newClusterCAScheme(), &testClusterRegSecret, &testMCTLSSecret)

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// Verify the CA secrets were not updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(origRegCA, localSecret.Data["ca-bundle"], "registration secret was changed")

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(origMCCA, adminSecret.Data["cacrt"], "MC CA secret was changed")
}

// TestSyncCACertsAreDifferent tests the synchronization method for the following use case.
// GIVEN a request to sync Admin CA certs
// WHEN the CAs are different
// THEN ensure that the secrets are updated.
func TestSyncCACertsAreDifferent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testAdminCASecret, err := getSampleClusterCASecret("testdata/clusterca-admincasecret-new.yaml")
	assert.NoError(err, "failed to read sample Admin CA Secret")

	testClusterRegSecret, err := getSampleClusterCASecret("testdata/clusterca-clusterregsecret.yaml")
	assert.NoError(err, "failed to read sample Cluster Registration Secret")

	testMCTLSSecret, err := getSampleClusterCASecret("testdata/clusterca-mctlssecret-new.yaml")
	assert.NoError(err, "failed to read sample MC TLS Secret")

	testMCCASecret, err := getSampleClusterCASecret("testdata/clusterca-mccasecret.yaml")
	assert.NoError(err, "failed to read sample MC CA Secret")

	testVMC, err := getSampleClusterCAVMC("testdata/clusterca-vmc.yaml")
	assert.NoError(err, "failed to read sample VMC")

	newRegCA := testAdminCASecret.Data["ca-bundle"]
	newMCCA := testMCTLSSecret.Data["ca.crt"]

	adminClient := fake.NewFakeClientWithScheme(newClusterCAScheme(), &testAdminCASecret, &testMCCASecret, &testVMC)

	localClient := fake.NewFakeClientWithScheme(newClusterCAScheme(), &testClusterRegSecret, &testMCTLSSecret)

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// Verify the CA secrets were not updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(newRegCA, localSecret.Data["ca-bundle"], "registration secret was changed")

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(newMCCA, adminSecret.Data["cacrt"], "MC CA secret was changed")
}

// getSampleClusterCASecret creates and returns a sample Secret used in tests
func getSampleClusterCASecret(filePath string) (corev1.Secret, error) {
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

// getSampleClusterCAVMC creates and returns a sample VMC
func getSampleClusterCAVMC(filePath string) (platformopclusters.VerrazzanoManagedCluster, error) {
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	sampleVMCFile, err := filepath.Abs(filePath)
	if err != nil {
		return vmc, err
	}

	rawResource, err := clusterstest.ReadYaml2Json(sampleVMCFile)
	if err != nil {
		return vmc, err
	}

	err = json.Unmarshal(rawResource, &vmc)
	return vmc, err
}

func newClusterCAScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	corev1.SchemeBuilder.AddToScheme(scheme)
	platformopclusters.AddToScheme(scheme)
	return scheme
}
