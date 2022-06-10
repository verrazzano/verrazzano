// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testConfigMapName            = "test-cm-name"
	testDeploymentName           = "test-deployment"
	testMetricsTemplateNamespace = "test-namespace"
	testMetricsTemplateName      = "test-template-name"
	testMetricsBindingNamespace  = "test-namespace"
	testMetricsBindingName       = "test-binding-name"
	deploymentKind               = "Deployment"
	deploymentGroup              = "apps"
	deploymentVersion            = "v1"
)

// TestCreateJobName tests the job name creator
// GIVEN a set of names
// WHEN the function is invoked
// THEN verify the name is correctly given
func TestCreateJobName(t *testing.T) {
	assert := asserts.New(t)
	assert.Equal(fmt.Sprintf("%s_%s_%s_%s_%s", testMetricsBindingNamespace, testDeploymentName, deploymentGroup, deploymentVersion, deploymentKind), createJobName(metricsBinding))
}

// TestGetConfigData tests the data retrieval from a
// GIVEN a set of names
// WHEN the function is invoked
// THEN verify the name is correctly given
func TestGetConfigData(t *testing.T) {
	assert := asserts.New(t)
	// Test normal config
	configMap, err := getConfigMapFromTestFile(true)
	assert.NoError(err, "Could not get test file")
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
func getConfigMapFromTestFile(empty bool) (*v1.ConfigMap, error) {
	var configMapData []byte
	var err error
	if empty {
		configMapData, err = os.ReadFile("./testdata/cmDataEmpty.yaml")
		if err != nil {
			return nil, err
		}
	} else {
		configMapData, err = os.ReadFile("./testdata/cmDataFilled.yaml")
		if err != nil {
			return nil, err
		}
	}

	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: vzconst.VerrazzanoSystemNamespace,
		},
		Data: map[string]string{
			prometheusConfigKey: string(configMapData),
		},
	}
	return &configMap, nil
}

// Returns a secret from the testdata file
func getSecretFromTestFile(empty bool) (*v1.Secret, error) {
	var secretData []byte
	var err error
	if empty {
		secretData, err = os.ReadFile("./testdata/secretDataEmpty.yaml")
		if err != nil {
			return nil, err
		}
	} else {
		secretData, err = os.ReadFile("./testdata/secretDataFilled.yaml")
		if err != nil {
			return nil, err
		}
	}

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconst.PrometheusOperatorNamespace,
			Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
		},
		Data: map[string][]byte{
			prometheusConfigKey: secretData,
		},
	}
	return &secret, nil
}

// Returns a Metrics Template from the testdata file
func getTemplateTestFile() (*vzapi.MetricsTemplate, error) {
	scrapeConfig, err := os.ReadFile("./testdata/scrape-config-template.yaml")
	if err != nil {
		return nil, err
	}
	template := metricsTemplate.DeepCopy()
	template.Spec.PrometheusConfig.ScrapeConfigTemplate = string(scrapeConfig)
	return template, nil
}
