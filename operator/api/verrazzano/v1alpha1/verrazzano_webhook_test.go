// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

const webhookTestValidChartYAML = `
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
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
	assert.Nil(t, currentSpec.ValidateCreate())
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
	assert.Nil(t, currentSpec.ValidateCreate())
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
	currentSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.7.0",
			Profile: "dev",
		},
	}
	assert.NotNil(t, currentSpec.ValidateCreate())
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
	assert.Nil(t, newSpec.ValidateUpdate(oldSpec))
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
	assert.Nil(t, newSpec.ValidateUpdate(oldSpec))
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
	assert.NotNil(t, newSpec.ValidateUpdate(oldSpec))
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
	assert.NotNil(t, newSpec.ValidateUpdate(oldSpec))
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
	assert.NotNil(t, newSpec.ValidateUpdate(oldSpec))
}

// TestDeleteCallbackSuccess Tests the create callback with valid spec version
func TestDeleteCallbackSuccess(t *testing.T) {
	oldSpec := &Verrazzano{
		Spec: VerrazzanoSpec{
			Version: "v0.6.0",
			Profile: "dev",
		},
	}
	assert.Nil(t, oldSpec.ValidateDelete())
}
