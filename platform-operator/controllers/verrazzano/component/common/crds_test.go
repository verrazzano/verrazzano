// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsApplyCRDYamlValid tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the yaml is valid
// THEN no error is returned
func TestIsApplyCRDYamlValid(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "../../../../helm_config"
	assert.Nil(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), config.GetHelmAppOpChartsDir()))
}

// TestIsApplyCRDYamlInvalidPath tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the path is invalid
// THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidPath(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "./testdata"
	assert.Error(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), ""))
}

// TestIsApplyCRDYamlInvalidChart tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the yaml is invalid
// THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidChart(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "invalidPath"
	assert.Error(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), ""))
}

// TestConvertVerrazzanoCR tests the ConvertVerrazzanoCR function
func TestConvertVerrazzanoCR(t *testing.T) {
	vz := vzapi.Verrazzano{}
	convertVZ := v1beta1.Verrazzano{}
	config.TestHelmConfigDir = "invalidPath"
	// GIVEN old Vz and a newer version of Vz
	// WHEN the ConvertVerrazzanoCR is called
	// THEN no error is returned
	err := ConvertVerrazzanoCR(&vz, &convertVZ)
	assert.Nil(t, err)

	// GIVEN a nil old Vz and a valid new Vz
	// WHEN the ConvertVerrazzanoCR is called
	// THEN an appropriate error is returned
	err = ConvertVerrazzanoCR(nil, &convertVZ)
	assert.Error(t, err)
}
