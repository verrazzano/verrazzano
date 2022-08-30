// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterRegSecretPath       = "testdata/clusterca-clusterregsecret.yaml"
	adminRegSecretPath         = "testdata/clusterca-adminregsecret.yaml"
	adminRegNewSecretPath      = "testdata/clusterca-adminregsecret-new.yaml"
	clusterCAAdminSecretPath   = "testdata/clusterca-admincasecret.yaml"
	mcCASecretPath             = "testdata/clusterca-mccasecret.yaml"
	vzTLSSecretPathNew         = "testdata/clusterca-mctlssecret-new.yaml"
	vzTLSSecretPath            = "testdata/clusterca-mctlssecret.yaml"
	vmcPath                    = "testdata/clusterca-vmc.yaml"
	sampleAdminCAReadErrMsg    = "failed to read sample Admin CA Secret"
	sampleClusterRegReadErrMsg = "failed to read sample Managed Cluster Registration Secret"
	sampleAdminRegReadErrMsg   = "failed to read sample Admin Cluster Registration Secret for the managed cluster"
	sampleMCTLSReadErrMsg      = "failed to read sample MC TLS Secret"
	sampleMCCAReadErrMsg       = "failed to read sample MC CA Secret"
	sampleVMCReadErrMsg        = "failed to read sample VMC"
	regSecChangedErrMsg        = "registration secret was changed"
	mcCASecChangedErrMsg       = "MC CA secret was changed"
)

// TestSyncAdminCANoDifference tests the synchronization method for the following use case.
// GIVEN a request to sync Admin registration info
// WHEN the CAs are the same and registration info is the same
// THEN ensure that no secret is updated.
func TestSyncCACertsNoDifference(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testAdminCASecret, err := getSampleSecret(clusterCAAdminSecretPath)
	assert.NoError(err, sampleAdminCAReadErrMsg)

	testAdminRegSecret, err := getSampleSecret(adminRegSecretPath)
	assert.NoError(err, sampleAdminRegReadErrMsg)

	testClusterRegSecret, err := getSampleSecret(clusterRegSecretPath)
	assert.NoError(err, sampleClusterRegReadErrMsg)

	testMCTLSSecret, err := getSampleSecret(vzTLSSecretPath)
	assert.NoError(err, sampleMCTLSReadErrMsg)

	testMCCASecret, err := getSampleSecret(mcCASecretPath)
	assert.NoError(err, sampleMCCAReadErrMsg)

	testVMC, err := getSampleClusterCAVMC(vmcPath)
	assert.NoError(err, sampleVMCReadErrMsg)

	origRegCA := testClusterRegSecret.Data[mcconstants.AdminCaBundleKey]
	origMCCA := testMCCASecret.Data[keyCaCrtNoDot]

	adminClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testAdminCASecret, &testMCCASecret, &testVMC, &testAdminRegSecret).
		Build()

	localClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testClusterRegSecret, &testMCTLSSecret).
		Build()

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	localClusterResult, err := s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// assert no update on local cluster
	assert.Equal(controllerutil.OperationResultNone, localClusterResult)

	// Verify the CA secrets were not updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(origRegCA, localSecret.Data[mcconstants.AdminCaBundleKey], regSecChangedErrMsg)

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(origMCCA, adminSecret.Data[keyCaCrtNoDot], mcCASecChangedErrMsg)

	// The registration info should not have been changed since the admin secret had the same info
	// as the existing managed cluster registration secret
	assertRegistrationInfoEqual(t, localSecret, testClusterRegSecret)
}

// TestSyncCACertsAreDifferent tests the synchronization method for the following use case.
// GIVEN a request to sync Admin CA certs
// WHEN the CAs are different but registration info is same,
// THEN ensure that the secrets are updated, but nothing else is
func TestSyncCACertsAreDifferent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testAdminCASecret, err := getSampleSecret("testdata/clusterca-admincasecret-new.yaml")
	assert.NoError(err, sampleAdminCAReadErrMsg)

	testAdminRegSecret, err := getSampleSecret(adminRegSecretPath)
	assert.NoError(err, sampleAdminRegReadErrMsg)

	testClusterRegSecret, err := getSampleSecret(clusterRegSecretPath)
	assert.NoError(err, sampleClusterRegReadErrMsg)

	testMCTLSSecret, err := getSampleSecret(vzTLSSecretPathNew)
	assert.NoError(err, sampleMCTLSReadErrMsg)

	testMCCASecret, err := getSampleSecret(mcCASecretPath)
	assert.NoError(err, sampleMCCAReadErrMsg)

	testVMC, err := getSampleClusterCAVMC(vmcPath)
	assert.NoError(err, sampleVMCReadErrMsg)

	newRegCA := testAdminCASecret.Data[mcconstants.AdminCaBundleKey]
	newMCCA := testMCTLSSecret.Data[mcconstants.CaCrtKey]

	adminClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testAdminCASecret, &testMCCASecret, &testVMC, &testAdminRegSecret).
		Build()

	localClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testClusterRegSecret, &testMCTLSSecret).
		Build()

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	localClusterResult, err := s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// assert there was a change on local cluster
	assert.NotEqual(controllerutil.OperationResultNone, localClusterResult)

	// Verify the CA secrets were updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(newRegCA, localSecret.Data[mcconstants.AdminCaBundleKey], regSecChangedErrMsg)

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(newMCCA, adminSecret.Data[keyCaCrtNoDot], mcCASecChangedErrMsg)

	// The registration info should not have been changed since the admin secret had the same info
	// as the existing managed cluster registration secret
	assertRegistrationInfoEqual(t, localSecret, testClusterRegSecret)
}

// Test the case when managed cluster uses Let's Encrypt staging (i.e. tls-ca-additional secret
// is present, and that should be preferred for sync even if verrazzano-tls is present.)
func TestSyncCACertsAdditionalTLSPresent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data
	testAdminCASecret, err := getSampleSecret("testdata/clusterca-admincasecret-new.yaml")
	assert.NoError(err, sampleAdminCAReadErrMsg)

	testAdminRegSecret, err := getSampleSecret(adminRegSecretPath)
	assert.NoError(err, sampleAdminRegReadErrMsg)

	testClusterRegSecret, err := getSampleSecret(clusterRegSecretPath)
	assert.NoError(err, sampleClusterRegReadErrMsg)

	// Managed cluster "normal" VZ ingress TLS secret (verrazzano-tls)
	testMCTLSSecret, err := getSampleSecret(vzTLSSecretPathNew)
	assert.NoError(err, sampleMCTLSReadErrMsg)

	// Managed cluster additional TLS secret is also present
	testMCAdditionalTLSSecret, err := getSampleSecret("testdata/clusterca-mc-additionaltls-secret.yaml")
	assert.NoError(err, "failed to read sample MC additional TLS CA Secret")

	testMCCASecret, err := getSampleSecret(mcCASecretPath)
	assert.NoError(err, sampleMCCAReadErrMsg)

	testVMC, err := getSampleClusterCAVMC(vmcPath)
	assert.NoError(err, sampleVMCReadErrMsg)

	newRegCA := testAdminCASecret.Data[mcconstants.AdminCaBundleKey]
	// Managed cluster additional TLS secret is the one to sync to admin cluster
	newMCCA := testMCAdditionalTLSSecret.Data[constants.AdditionalTLSCAKey]

	adminClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testAdminCASecret, &testMCCASecret, &testVMC, &testAdminRegSecret).
		Build()

	localClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testClusterRegSecret, &testMCTLSSecret, &testMCAdditionalTLSSecret).
		Build()

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	localClusterResult, err := s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// assert there was a change on local cluster
	assert.NotEqual(controllerutil.OperationResultNone, localClusterResult)

	// Verify the CA secrets were updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(newRegCA, localSecret.Data[mcconstants.AdminCaBundleKey], regSecChangedErrMsg)

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(newMCCA, adminSecret.Data[keyCaCrtNoDot], "MC CA secret on admin cluster did not match the additional TLS CA secret on managed cluster.")

	// Registration info should not have changed
	assertRegistrationInfoEqual(t, localSecret, testClusterRegSecret)
}

// TestSyncRegistrationInfoDifferent tests the synchronization method for the following use case.
// GIVEN a request to sync Admin registration info
// WHEN the registration info is different but CAs are the same,
// THEN ensure that the reg info is updated, but nothing else is
func TestSyncRegistrationInfoDifferent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Test data

	// Admin CA secret is the unchanged one
	testAdminCASecret, err := getSampleSecret(clusterCAAdminSecretPath)
	assert.NoError(err, sampleAdminCAReadErrMsg)

	// Use the "updated" admin registration data to simulate admin cluster reg secret changed
	testAdminRegSecret, err := getSampleSecret(adminRegNewSecretPath)
	assert.NoError(err, sampleAdminRegReadErrMsg)

	testClusterRegSecret, err := getSampleSecret(clusterRegSecretPath)
	assert.NoError(err, sampleClusterRegReadErrMsg)

	testMCCASecret, err := getSampleSecret(mcCASecretPath)
	assert.NoError(err, sampleMCCAReadErrMsg)

	testMCTLSSecret, err := getSampleSecret(vzTLSSecretPath)
	assert.NoError(err, sampleMCTLSReadErrMsg)

	testVMC, err := getSampleClusterCAVMC(vmcPath)
	assert.NoError(err, sampleVMCReadErrMsg)

	origRegCA := testClusterRegSecret.Data[mcconstants.AdminCaBundleKey]
	origMCCA := testMCCASecret.Data[keyCaCrtNoDot]
	newRegSecret := testAdminRegSecret

	adminClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testAdminCASecret, &testMCCASecret, &testVMC, &testAdminRegSecret).
		Build()

	localClient := fake.NewClientBuilder().
		WithScheme(newClusterCAScheme()).
		WithRuntimeObjects(&testClusterRegSecret, &testMCTLSSecret).
		Build()

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	localClusterResult, err := s.syncClusterCAs()

	// Validate the results
	assert.NoError(err)

	// assert there was a change on local cluster
	assert.NotEqual(controllerutil.OperationResultNone, localClusterResult)

	// Verify the CA secrets were NOT updated
	localSecret := &corev1.Secret{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testClusterRegSecret.Name, Namespace: testClusterRegSecret.Namespace}, localSecret)
	assert.NoError(err)
	assert.Equal(origRegCA, localSecret.Data[mcconstants.AdminCaBundleKey], regSecChangedErrMsg)

	adminSecret := &corev1.Secret{}
	err = s.AdminClient.Get(s.Context, types.NamespacedName{Name: testMCCASecret.Name, Namespace: testMCCASecret.Namespace}, adminSecret)
	assert.NoError(err)
	assert.Equal(origMCCA, adminSecret.Data[keyCaCrtNoDot], mcCASecChangedErrMsg)

	// The registration info SHOULD have been changed since the admin secret had different info
	// from the existing managed cluster registration secret
	assertRegistrationInfoEqual(t, localSecret, newRegSecret)
}

func TestSyncRegistrationFromAdminCluster(t *testing.T) {
	testAdminCASecret, err := getSampleSecret(clusterCAAdminSecretPath)
	asserts.NoError(t, err, sampleAdminCAReadErrMsg)
	log := zap.S().With("test")
	tests := []struct {
		name                    string
		testAdminCASecret       *corev1.Secret
		adminRegistrationSecret *corev1.Secret
		localRegistrationSecret *corev1.Secret
		expectedOperation       controllerutil.OperationResult
		expectedError           error
	}{
		{
			"OS url is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.ESURLKey: "new OS url",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"OS CA bundle is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.ESCaBundleKey: "new CA bundle",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Registration username is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.RegistrationUsernameKey: "new user",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Registration password  is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.RegistrationPasswordKey: "new password",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Keycloak url is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.KeycloakURLKey: "new keycloak url",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger OS url is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSURLKey: "new value",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger OS username is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSUsernameKey: "newuser",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger OS password is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSPasswordKey: "newpassword",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger TLS CA is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSTLSCAKey: "newCAKey",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger TLS cert is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSTLSCertKey: "newTLSCertKey",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Jaeger TLS key is updated in admin cluster but not synced to managed1",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, map[string]string{
				mcconstants.JaegerOSTLSKey: "newTLSKey",
			}),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"Admin CA bundle is different in managed cluster",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, nil),
			createSecretWithOverrides(clusterRegSecretPath, map[string]string{
				mcconstants.AdminCaBundleKey: "new CA bundle",
			}),
			controllerutil.OperationResultUpdated,
			nil,
		},
		{
			"All values are in sync between admin and managed1 cluster",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, nil),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultNone,
			nil,
		},
		{
			"When registration secret is missing in admin cluster, then it should return error",
			&testAdminCASecret,
			nil,
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultNone,
			fmt.Errorf("secrets \"verrazzano-cluster-managed1-registration\" not found"),
		},
		{
			"When registration secret is missing in local cluster, then it should return error",
			&testAdminCASecret,
			createSecretWithOverrides(adminRegSecretPath, nil),
			nil,
			controllerutil.OperationResultNone,
			fmt.Errorf("secrets \"verrazzano-cluster-registration\" not found"),
		},
		{
			"When CA cert secret is missing in admin cluster, then it should return error",
			nil,
			createSecretWithOverrides(adminRegSecretPath, nil),
			createSecretWithOverrides(clusterRegSecretPath, nil),
			controllerutil.OperationResultNone,
			fmt.Errorf("secrets \"verrazzano-local-ca-bundle\" not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminRuntimeObjects := []runtime.Object{}
			if tt.testAdminCASecret != nil {
				adminRuntimeObjects = append(adminRuntimeObjects, tt.testAdminCASecret)
			}
			if tt.adminRegistrationSecret != nil {
				adminRuntimeObjects = append(adminRuntimeObjects, tt.adminRegistrationSecret)
			}
			adminClient := fake.NewClientBuilder().
				WithScheme(newClusterCAScheme()).
				WithRuntimeObjects(adminRuntimeObjects...).
				Build()

			localRuntimeObjects := []runtime.Object{}
			if tt.localRegistrationSecret != nil {
				localRuntimeObjects = append(localRuntimeObjects, tt.localRegistrationSecret)
			}
			localClient := fake.NewClientBuilder().
				WithScheme(newClusterCAScheme()).
				WithRuntimeObjects(localRuntimeObjects...).
				Build()

			s := &Syncer{
				AdminClient:        adminClient,
				LocalClient:        localClient,
				Log:                log,
				ManagedClusterName: testClusterName,
				Context:            context.TODO(),
			}
			actualOperationResult, err := s.syncRegistrationFromAdminCluster()
			if tt.expectedError != nil {
				asserts.Equal(t, err.Error(), tt.expectedError.Error())
				return
			}
			asserts.NoError(t, err)
			asserts.Equal(t, tt.expectedOperation, actualOperationResult)
			// post sync call both the secrets should have the same values of registration secrets
			// and calling sync again should be a no-op (unchanged).
			reSyncOperationResult, err := s.syncRegistrationFromAdminCluster()
			asserts.NoError(t, err)
			asserts.Equal(t, controllerutil.OperationResultNone, reSyncOperationResult)
		})
	}
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

func assertRegistrationInfoEqual(t *testing.T, regSecret1 *corev1.Secret, regSecret2 corev1.Secret) {
	asserts.Equal(t, regSecret1.Data[mcconstants.ESURLKey], regSecret2.Data[mcconstants.ESURLKey], "ES URL is different")
	asserts.Equal(t, regSecret1.Data[mcconstants.KeycloakURLKey], regSecret2.Data[mcconstants.KeycloakURLKey], "Keycloak URL is different")
	asserts.Equal(t, regSecret1.Data[mcconstants.RegistrationUsernameKey], regSecret2.Data[mcconstants.RegistrationUsernameKey], "Registration Username is different")
	asserts.Equal(t, regSecret1.Data[mcconstants.RegistrationPasswordKey], regSecret2.Data[mcconstants.RegistrationPasswordKey], "Registration Password is different")
	asserts.Equal(t, regSecret1.Data[mcconstants.ESCaBundleKey], regSecret2.Data[mcconstants.ESCaBundleKey], "Registration Password is different")
}

func createSecretWithOverrides(filepath string, overrides map[string]string) *corev1.Secret {
	secret, err := getSampleSecret(filepath)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return &corev1.Secret{}
	}
	for key, value := range overrides {
		secret.Data[key] = []byte(value)
	}
	return &secret
}
