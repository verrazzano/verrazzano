// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
	verrazzanoRepo  = "ghcr.io/verrazzano"
	oracleRepo      = "ghcr.io/oracle"
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

// TestGetImageOverrides tests the getImageOverrides function
// GIVEN a call to getImageOverrides
//
//	WHEN a test Verrazzano bom is supplied
//	THEN the template inputs for generating a clusterctl.yaml file are returned
func TestGetImageOverrides(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	templateInput, err := getImageOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, templateInput)
	assert.Equal(t, "v1.3.3", templateInput.APIVersion)
	assert.Equal(t, verrazzanoRepo, templateInput.APIRepository)
	assert.Equal(t, "v1.3.3-20230427222746-876fe3dc9", templateInput.APITag)
	assert.Equal(t, "v0.8.1", templateInput.OCIVersion)
	assert.Equal(t, oracleRepo, templateInput.OCIRepository)
	assert.Equal(t, "v0.8.1", templateInput.OCITag)
	assert.Equal(t, "v0.1.0", templateInput.OCNEBootstrapVersion)
	assert.Equal(t, verrazzanoRepo, templateInput.OCNEBootstrapRepository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", templateInput.OCNEBootstrapTag)
	assert.Equal(t, "v0.1.0", templateInput.OCNEControlPlaneVersion)
	assert.Equal(t, verrazzanoRepo, templateInput.OCNEControlPlaneRepository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", templateInput.OCNEControlPlaneTag)
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
	templateInput, err := getImageOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, templateInput)
	clusterctl, err := applyTemplate(clusterctlYamlTemplate, templateInput)
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
