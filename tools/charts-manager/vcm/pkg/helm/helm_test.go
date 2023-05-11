// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
)

const (
	testData         = "testdata"
	testChart        = "testChart"
	testVersion      = "x.y.z"
	testChartsDir    = "testdata/charts"
	testRepoConfig   = testData + "/repositories.yaml"
	testRepoCache    = testData + "/repocache"
	envVarRepoConfig = "HELM_REPOSITORY_CONFIG"
	envVarRepoCache  = "HELM_REPOSITORY_CACHE"
)

var prevRepoConfig = os.Getenv(envVarRepoConfig)
var prevRepoCache = os.Getenv(envVarRepoCache)

// TestNewHelmConfig tests that function NewHelmConfig succeeds for default inputs
// GIVEN a call to NewHelmConfig
//
//	WHEN helm repo env variables are correctly set
//	THEN the NewHelmConfig returns a valid HelmConfig.
func TestNewHelmConfig(t *testing.T) {
	defer helmCleanUp()
	helmConfig := getHelmConfig(t)
	assert.Equal(t, testRepoConfig, helmConfig.settings.RepositoryConfig)
	assert.Equal(t, testRepoCache, helmConfig.settings.RepositoryCache)
	assert.NotNil(t, helmConfig.helmRepoFile)
}

func getHelmConfig(t *testing.T) *VerrazzanoHelmConfig {
	os.Setenv(envVarRepoConfig, testRepoConfig)
	os.Setenv(envVarRepoCache, testRepoCache)
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	helmConfig, err := NewHelmConfig(rc)
	assert.NoError(t, err)
	assert.NotNil(t, helmConfig)
	return helmConfig
}

func helmCleanUp() {
	os.Setenv(envVarRepoConfig, prevRepoConfig)
	os.Setenv(envVarRepoCache, prevRepoCache)
}
