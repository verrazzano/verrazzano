// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
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
