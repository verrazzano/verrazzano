// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	goerrors "errors"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateCallbackSuccessWithVersion Tests the create callback with valid spec version
// GIVEN a ValidateCreate() request with a valid version
// WHEN the version provided is a valid version
// THEN no error is returned
func TestCreateCallbackSuccessWithVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.1.0",
			Profile: "dev",
		},
	}
	assert.NoError(t, currentSpec.ValidateCreate())
}

// TestCreateCallbackSuccessWithoutVersion Tests the create callback with no spec version
// GIVEN a ValidateCreate() request with a valid version
// WHEN no version is provided
// THEN no error is returned
func TestCreateCallbackSuccessWithoutVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
	}
	assert.NoError(t, currentSpec.ValidateCreate())
}

// TestCreateCallbackFailsWithInvalidVersion Tests the create callback with invalid spec version
// GIVEN a ValidateCreate() request with an invalid version
// WHEN an invalid version is provided
// THEN an error is returned
func TestCreateCallbackFailsWithInvalidVersion(t *testing.T) {
	assert.Error(t, runCreateCallbackWithInvalidVersion(t))
}

// TestCreateCallbackWithInvalidVersionValidationDisabled Tests the create callback with invalid spec version passes with validation disabled
// GIVEN a ValidateCreate() request
// WHEN an invalid version is provided and webhook validation is disabled
// THEN no error is returned
func TestCreateCallbackWithInvalidVersionValidationDisabled(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{WebhookValidationEnabled: false})
	assert.NoError(t, runCreateCallbackWithInvalidVersion(t))
}

// runCreateCallbackWithInvalidVersion Shared test impl for cases with/without validation enbabled
func runCreateCallbackWithInvalidVersion(t *testing.T) error {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0180,
			Profile: "dev",
		},
	}
	err := currentSpec.ValidateCreate()
	return err
}

// TestUpdateCallbackSuccessWithNewVersion Tests the update callback with valid spec version at the same bom revision
// GIVEN a ValidateUpdate() request
// WHEN a valid version is provided and is at the same bom value
// THEN no error is returned
func TestUpdateCallbackSuccessWithNewVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v110,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.1.0",
			Profile: "dev",
		},
	}

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	assert.NoError(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackSuccessWithNewVersion Tests the update callback with valid spec versions in both
// GIVEN a ValidateUpdate() request
// WHEN valid versions exist in both specs, and the new version > old version
// THEN no error is returned
func TestUpdateCallbackSuccessWithOldAndNewVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.16.0",
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: "v0.16.0",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.1.0",
			Profile: "dev",
		},
	}

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	assert.NoError(t, newSpec.ValidateUpdate(oldSpec))
}

// TestRollbackRejected Tests the update callback with valid spec versions in both
// GIVEN a ValidateUpdate() request
// WHEN no upgrade happened before (current spec ver empty), BOM ver is older than installed ver, and newSpec ver is older version
// THEN no error is returned
func TestRollbackRejected(t *testing.T) {
	config.SetDefaultBomFilePath(testRollbackBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v110,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v100,
			Profile: "dev",
		},
	}

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	err := newSpec.ValidateUpdate(oldSpec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rollback is not supported")
}

// TestUpdateCallbackFailsWithOldGreaterThanNewVersion Tests the create callback with old version > new
// GIVEN a ValidateUpdate() request
// WHEN valid versions exist in both specs, and the new old > new version
// THEN an error is returned
func TestUpdateCallbackFailsWithOldGreaterThanNewVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.2.0",
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: "v1.2.0",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.1.0",
			Profile: "dev",
		},
	}
	assert.Error(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackFailsWithInvalidNewVersion Tests the create callback with invalid new version
// GIVEN a ValidateUpdate() request
// WHEN the new version is valid but not the same as the bom version
// THEN an error is returned
func TestUpdateCallbackFailsWithInvalidNewVersion(t *testing.T) {
	assert.Error(t, runUpdateWithInvalidVersionTest(t))
}

// TestUpdateCallbackFailsWithInvalidNewVersion Tests the create callback with invalid new version fails
// GIVEN a ValidateUpdate() request
// WHEN an invalid version is provided and webhook validation is disabled
// THEN no error is returned
func TestUpdateCallbackWithInvalidNewVersionValidationDisabled(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{WebhookValidationEnabled: false})
	assert.NoError(t, runUpdateWithInvalidVersionTest(t))
}

// runUpdateWithInvalidVersionTest Shared test logic for update with invalid version
func runUpdateWithInvalidVersionTest(t *testing.T) error {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
		Status: VerrazzanoStatus{
			Version: v0160,
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: v0180,
			Profile: "dev",
		},
	}
	return newSpec.ValidateUpdate(oldSpec)
}

// TestUpdateCallbackFailsChangeProfile Tests the create callback with a changed profile
// GIVEN a ValidateUpdate() request
// WHEN the profile is changed
// THEN an error is returned
func TestUpdateCallbackFailsChangeProfile(t *testing.T) {
	assert.Error(t, runUpdateCallbackChangedProfileTest())
}

// TestUpdateCallbackChangeProfileValidationDisabled Tests the create callback with a changed profile passes with validation disabled
// GIVEN a ValidateUpdate() request
// WHEN the profile is changed and webhook validation is disabled
// THEN no error is returned
func TestUpdateCallbackChangeProfileValidationDisabled(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{WebhookValidationEnabled: false})
	assert.NoError(t, runUpdateCallbackChangedProfileTest())
}

// runUpdateCallbackChangedProfileTest Shared test logic for update with changed profile
func runUpdateCallbackChangedProfileTest() error {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "prod",
		},
	}
	err := newSpec.ValidateUpdate(oldSpec)
	return err
}

// TestDeleteCallbackSuccess Tests the create callback with valid spec version
// GIVEN a ValidateDelete() request
// WHEN
// THEN no error is returned
func TestDeleteCallbackSuccess(t *testing.T) {
	assert.NoError(t, runDeleteCallbackTest())
}

// TestDeleteCallbackDisabled Tests the create callback with valid spec version; largely for code coverage right now
// GIVEN a ValidateDelete() request
// WHEN webhook validation is disabled
// THEN no error is returned
func TestDeleteCallbackDisabled(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{WebhookValidationEnabled: false})
	assert.NoError(t, runDeleteCallbackTest())
}

// runDeleteCallbackTest shared logic for ValidateDelete tests
func runDeleteCallbackTest() error {
	deletedSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v1.1.0",
			Profile: "dev",
		},
	}
	return deletedSpec.ValidateDelete()
}

// Test_verifyPlatformOperatorSingleton Tests the verifyPlatformOperatorSingleton check
// GIVEN a verifyPlatformOperatorSingleton call
// WHEN more than one Pod matches the selection criteria
// THEN an error is returned
func Test_verifyPlatformOperatorSingleton(t *testing.T) {
	vz := &Verrazzano{}

	labels := map[string]string{
		"app": "verrazzano-platform-operator",
	}
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(), &v1.PodList{
			TypeMeta: metav1.TypeMeta{},
			Items: []v1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: constants.VerrazzanoInstallNamespace, Labels: labels}},
				{ObjectMeta: metav1.ObjectMeta{Name: "thud", Namespace: constants.VerrazzanoInstallNamespace, Labels: labels}},
			},
		}), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	assert.Error(t, vz.verifyPlatformOperatorSingleton())
}

// Test_verifyPlatformOperatorSingletonNoMatchingLabels Tests the verifyPlatformOperatorSingleton check
// GIVEN a verifyPlatformOperatorSingleton call
// WHEN no Pods match the selection criteria
// THEN no error is returned
func Test_verifyPlatformOperatorSingletonNoMatchingLabels(t *testing.T) {
	vz := &Verrazzano{}

	labels := map[string]string{
		"app": "someapp",
	}
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(), &v1.PodList{
			TypeMeta: metav1.TypeMeta{},
			Items: []v1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: constants.VerrazzanoInstallNamespace, Labels: labels}},
			},
		}), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	assert.NoError(t, vz.verifyPlatformOperatorSingleton())
}

// Test_verifyPlatformOperatorSingletonSuccess Tests the verifyPlatformOperatorSingleton check
// GIVEN a verifyPlatformOperatorSingleton call
// WHEN only one Pod matches the selection criteria
// THEN no error is returned
func Test_verifyPlatformOperatorSingletonSuccess(t *testing.T) {
	vz := &Verrazzano{}

	labels := map[string]string{
		"app": "verrazzano-platform-operator",
	}
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(), &v1.PodList{
			TypeMeta: metav1.TypeMeta{},
			Items: []v1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: constants.VerrazzanoInstallNamespace, Labels: labels}},
			},
		}), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	assert.NoError(t, vz.verifyPlatformOperatorSingleton())
}

// Test_combineErrors Tests combineErrors
// GIVEN slices of errors
// WHEN there is one or more errors
// THEN a combined error is returned
func Test_combineErrors(t *testing.T) {
	var errs []error
	err := validators.CombineErrors(errs)
	assert.Nil(t, err)

	errs = []error{goerrors.New("e1")}
	err = validators.CombineErrors(errs)
	assert.NotNil(t, err)
	assert.Equal(t, "e1", err.Error())

	errs = []error{goerrors.New("e1"), goerrors.New("e2"), goerrors.New("e3")}
	err = validators.CombineErrors(errs)
	assert.NotNil(t, err)
	assert.Equal(t, "[e1, e2, e3]", err.Error())
}

// TestUpdateMissingOciLoggingApiSecret Tests the update callback with valid spec config with oci-logging
// GIVEN a ValidateUpdate() request
// WHEN a new CR contains oci-logging with apiSecret
// THEN validation error is returned
func TestUpdateMissingOciLoggingApiSecret(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	oldSpec := &Verrazzano{Spec: VerrazzanoSpec{Profile: "dev"}}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
			Components: ComponentSpec{
				Fluentd: &FluentdComponent{
					OCI: &OciLoggingConfiguration{
						DefaultAppLogID: "testDefaultAppLogID",
						SystemLogID:     "DefaultAppLogID",
						APISecret:       "testAPISecret",
					},
				},
			},
		},
	}
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() {
		config.SetDefaultBomFilePath("")
		getControllerRuntimeClient = validators.GetClient
	}()
	assert.Error(t, newSpec.ValidateUpdate(oldSpec))
}
