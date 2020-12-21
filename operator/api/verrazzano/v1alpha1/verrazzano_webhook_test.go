// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/operator/internal/util/env"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const webhookTestValidChartYAML = `
apiVersion: v1
description: A Helm chart for Verrazzano
name: verrazzano
version: 0.6.0
appVersion: 0.6.0
`

// TestCreateCallbackSuccessWithVersion Tests the create callback with valid spec version
// GIVEN a ValidateCreate() request with a valid version
// WHEN the version provided is a valid version
// THEN no error is returned
func TestCreateCallbackSuccessWithVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
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
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

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
	os.Setenv(env.WebHookValidationEnabled, "false")
	defer os.Unsetenv(env.WebHookValidationEnabled)
	assert.NoError(t, runCreateCallbackWithInvalidVersion(t))
}

// runCreateCallbackWithInvalidVersion Shared test impl for cases with/without validation enbabled
func runCreateCallbackWithInvalidVersion(t *testing.T) error {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.7.0",
			Profile: "dev",
		},
	}
	err := currentSpec.ValidateCreate()
	return err
}

// TestUpdateCallbackSuccessWithNewVersion Tests the update callback with valid spec version at the same chart revision
// GIVEN a ValidateUpdate() request
// WHEN a valid version is provided and is at the same chart value
// THEN no error is returned
func TestUpdateCallbackSuccessWithNewVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.NoError(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackSuccessWithNewVersion Tests the create callback with valid spec versions in both
// GIVEN a ValidateUpdate() request
// WHEN valid versions exist in both specs, and the new version > old version
// THEN no error is returned
func TestUpdateCallbackSuccessWithOldAndNewVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.5.0",
			Profile: "dev",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.NoError(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackFailsWithOldGreaterThanNewVersion Tests the create callback with old version > new
// GIVEN a ValidateUpdate() request
// WHEN valid versions exist in both specs, and the new old > new version
// THEN an error is returned
func TestUpdateCallbackFailsWithOldGreaterThanNewVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.8.0",
			Profile: "dev",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.Error(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackFailsWithInvalidNewVersion Tests the create callback with invalid new version
// GIVEN a ValidateUpdate() request
// WHEN the new version is valid but not the same as the chart version
// THEN an error is returned
func TestUpdateCallbackFailsWithInvalidNewVersion(t *testing.T) {
	assert.Error(t, runUpdateWithInvalidVersionTest(t))
}

// TestUpdateCallbackFailsWithInvalidNewVersion Tests the create callback with invalid new version fails
// GIVEN a ValidateUpdate() request
// WHEN an invalid version is provided and webhook validation is disabled
// THEN no error is returned
func TestUpdateCallbackWithInvalidNewVersionValidationDisabled(t *testing.T) {
	os.Setenv(env.WebHookValidationEnabled, "false")
	defer os.Unsetenv(env.WebHookValidationEnabled)
	assert.NoError(t, runUpdateWithInvalidVersionTest(t))
}

// runUpdateWithInvalidVersionTest Shared test logic for update with invalid version
func runUpdateWithInvalidVersionTest(t *testing.T) error {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
	}
	newSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.7.0",
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
	os.Setenv(env.WebHookValidationEnabled, "false")
	defer os.Unsetenv(env.WebHookValidationEnabled)
	assert.NoError(t, runUpdateCallbackChangedProfileTest())
}

// runUpdateCallbackChangedProfileTest Shared test logic for update with changed profile
func runUpdateCallbackChangedProfileTest() error {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
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
	os.Setenv(env.WebHookValidationEnabled, "false")
	defer os.Unsetenv(env.WebHookValidationEnabled)
	assert.NoError(t, runDeleteCallbackTest())
}

// runDeleteCallbackTest shared logic for ValidateDelete tests
func runDeleteCallbackTest() error {
	deletedSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	return deletedSpec.ValidateDelete()
}
