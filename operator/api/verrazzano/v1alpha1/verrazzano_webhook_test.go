// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/operator/internal/util/env"
	"io/ioutil"
	"os"
	"testing"
)

const webhookTestValidChartYAML = `
apiVersion: v1
description: A Helm chart for Verrazzano
name: verrazzano
version: 0.6.0
appVersion: 0.6.0
`

// TestCreateCallbackSuccessWithVersion Tests the create callback with valid spec version
func TestCreateCallbackSuccessWithVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.NoError(t, currentSpec.ValidateCreate())
}

// TestCreateCallbackSuccessWithoutVersion Tests the create callback with no spec version
func TestCreateCallbackSuccessWithoutVersion(t *testing.T) {
	chartYaml := webhookTestValidChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Profile: "dev",
		},
	}
	assert.NoError(t, currentSpec.ValidateCreate())
}

// TestCreateCallbackFailsWithInvalidVersion Tests the create callback with invalid spec version
func TestCreateCallbackFailsWithInvalidVersion(t *testing.T) {
	assert.Error(t, runCreateCallbackWithInvalidVersion(t))
}

// TestCreateCallbackWithInvalidVersionValidationDisabled Tests the create callback with invalid spec version passes with validation disabled
func TestCreateCallbackWithInvalidVersionValidationDisabled(t *testing.T) {
	os.Setenv(env.DisableWebHookValidation, "true")
	defer os.Unsetenv(env.DisableWebHookValidation)
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
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.7.0",
			Profile: "dev",
		},
	}
	err := currentSpec.ValidateCreate()
	return err
}

// TestUpdateCallbackSuccessWithNewVersion Tests the create callback with valid spec version
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
func TestUpdateCallbackFailsWithInvalidNewVersion(t *testing.T) {
	assert.Error(t, runUpdateWithInvalidVersionTest(t))
}

// TestUpdateCallbackFailsWithInvalidNewVersion Tests the create callback with invalid new version fails
func TestUpdateCallbackWithInvalidNewVersionValidationDisabled(t *testing.T) {
	os.Setenv(env.DisableWebHookValidation, "true")
	defer os.Unsetenv(env.DisableWebHookValidation)
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
func TestUpdateCallbackFailsChangeProfile(t *testing.T) {
	assert.Error(t, runUpdateCallbackChangedProfileTest())
}

// TestUpdateCallbackChangeProfileValidationDisabled Tests the create callback with a changed profile passes with validation disabled
func TestUpdateCallbackChangeProfileValidationDisabled(t *testing.T) {
	os.Setenv(env.DisableWebHookValidation, "true")
	defer os.Unsetenv(env.DisableWebHookValidation)
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
func TestDeleteCallbackSuccess(t *testing.T) {
	assert.NoError(t, runDeleteCallbackTest())
}

// TestDeleteCallbackDisabled Tests the create callback with valid spec version; largely for code coverage right now
func TestDeleteCallbackDisabled(t *testing.T) {
	os.Setenv(env.DisableWebHookValidation, "true")
	defer os.Unsetenv(env.DisableWebHookValidation)
	assert.NoError(t, runDeleteCallbackTest())
}

func runDeleteCallbackTest() error {
	deletedSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	return deletedSpec.ValidateDelete()
}