// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
)

// TestSetEnvVariables tests the setEnvVariables function
// GIVEN a call to setEnvVariables
//
//	WHEN all env variables are set to the correct values
//	THEN true is returned
func TestSetEnvVariables(t *testing.T) {
	err := setEnvVariables()
	assert.Equal(t, "false", os.Getenv(initOCIClientsOnStartup))
	assert.Equal(t, "true", os.Getenv(expClusterResourceSet))
	assert.Equal(t, "true", os.Getenv(expMachinePool))
	assert.Equal(t, "true", os.Getenv(clusterTopology))
	assert.NoError(t, err)
}

// TestApplyTemplate tests the applyTemplate function
// GIVEN a call to applyTemplate
//
//	WHEN the template input is supplied
//	THEN a buffer containing the contents of clusterctl.yaml is returned and all parameters replaced
func TestApplyTemplate(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	clusterctl, err := applyTemplate(clusterctlYamlTemplate, overrides)
	assert.NoError(t, err)
	assert.NotEmpty(t, clusterctl)
	assert.NotContains(t, clusterctl.String(), "{{.")
}

// TestCreateClusterctlYaml tests the createClusterctlYaml function
// GIVEN a call to createClusterctlYaml
//
//	WHEN overrides from the BOM are applied
//	THEN a clusterctl.yaml file is created
func TestCreateClusterctlYaml(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	dir := os.TempDir() + "/" + time.Now().Format("20060102150405")
	setClusterAPIDir(dir)
	defer resetClusterAPIDir()
	defer os.RemoveAll(dir)
	err := createClusterctlYaml(compContext)
	assert.NoError(t, err)
	_, err = os.Stat(dir + "/clusterctl.yaml")
	assert.NoError(t, err)
}
