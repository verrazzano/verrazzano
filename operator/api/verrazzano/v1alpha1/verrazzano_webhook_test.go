// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
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
func TestCreateCallbackFailsWithInvalidVersion(t *testing.T) {
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
	assert.Error(t, currentSpec.ValidateCreate())
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
	assert.Error(t, newSpec.ValidateUpdate(oldSpec))
}

// TestUpdateCallbackFailsChangeProfile Tests the create callback with a changed profile
func TestUpdateCallbackFailsChangeProfile(t *testing.T) {
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
	assert.Error(t, newSpec.ValidateUpdate(oldSpec))
}

// TestDeleteCallbackSuccess Tests the create callback with valid spec version
func TestDeleteCallbackSuccess(t *testing.T) {
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.NoError(t, oldSpec.ValidateDelete())
}

// TestGetClient checks that we can get a controller runtime client
// GIVEN a controller runtime
// THEN ensure an error is not returned when getting a controller runtime client
func TestGetClient(t *testing.T) {
	client, err := getClient()
	assert.NotNil(t, client)
	assert.NoError(t, err)
}
