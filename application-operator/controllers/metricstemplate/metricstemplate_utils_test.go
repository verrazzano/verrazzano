// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testConfigMapName             = "test-cm-name"
	testDeploymentNamespace       = "test-namespace"
	testDeploymentName            = "test-deployment"
	testExistsDeploymentNamespace = "update-ns"
	testExistsDeploymentName      = "update-deployment"

	testCMUID               = "testCMUID"
	testMTUID               = "testMTUID"
	testDeploymentUID       = "testDeploymentUID"
	testExistsDeploymentUID = "updateUID"
)

// TestCreateJobName tests the job name creator
// GIVEN a set of names
// WHEN the function is invoked
// THEN verify the name is correctly given
func TestCreateJobName(t *testing.T) {
	assert := asserts.New(t)
	assert.Equal("__", createJobName(types.NamespacedName{Namespace: "", Name: ""}, ""))
	assert.Equal("1_2_3", createJobName(types.NamespacedName{Namespace: "1", Name: "2"}, "3"))
	assert.Equal("test-namespace_test-name_test-UID", createJobName(types.NamespacedName{Namespace: "test-namespace", Name: "test-name"}, "test-UID"))
}

// TestGetConfigData tests the data retrieval from a
// GIVEN a set of names
// WHEN the function is invoked
// THEN verify the name is correctly given
func TestGetConfigData(t *testing.T) {
	assert := asserts.New(t)
	// Test normal config
	configMap, err := getConfigMapFromTestFile()
	assert.NoError(err, "Could not get test file prometheus.yml")
	config, err := getConfigData(configMap)
	assert.NoError(err, "Could not create ConfigMap data")
	assert.NotNil(config)

	// Test empty config
	configMap = &v1.ConfigMap{Data: map[string]string{"prometheus.yml": ""}}
	config, err = getConfigData(configMap)
	assert.NoError(err, "Could not create empty ConfigMap data")
	assert.NotNil(config)

	// Test data does not exist
	configMap = &v1.ConfigMap{}
	config, err = getConfigData(configMap)
	assert.Error(err, "Expected error from nil Data")
	assert.Nil(config)
}

// Returns a configmap from the testdata file
func getConfigMapFromTestFile() (*v1.ConfigMap, error) {
	configmapData, err := os.ReadFile("./testdata/prometheus.yml")
	if err != nil {
		return nil, err
	}
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
		Data: map[string]string{
			prometheusConfigKey: string(configmapData),
		},
	}
	return &configMap, nil
}
