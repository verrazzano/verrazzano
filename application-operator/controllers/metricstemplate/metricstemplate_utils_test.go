// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"fmt"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	k8sapps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testConfigMapName             = "test-cm-name"
	testDeploymentNamespace       = "test-namespace"
	testDeploymentName            = "test-deployment"
	testExistsDeploymentNamespace = "update-ns"
	testExistsDeploymentName      = "update-deployment"
	testMetricsTemplateNamespace  = "test-namespace"
	testMetricsTemplateName       = "test-template-name"
	testMetricsBindingNamespace   = "test-namespace"
	testMetricsBindingName        = "test-binding-name"
	deploymentKind                = "Deployment"
)

var metricsBinding = v1alpha1.MetricsBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testMetricsBindingName,
		OwnerReferences: []metav1.OwnerReference{
			{
				Name: testDeploymentName,
				Kind: deploymentKind,
			},
		},
	},
	Spec: v1alpha1.MetricsBindingSpec{
		MetricsTemplate: v1alpha1.NamespaceName{
			Namespace: testMetricsTemplateNamespace,
			Name:      testMetricsTemplateName,
		},
		PrometheusConfigMap: v1alpha1.NamespaceName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      testConfigMapName,
		},
	},
}

var metricsTemplate = v1alpha1.MetricsTemplate{
	TypeMeta: metav1.TypeMeta{
		Kind:       metricsTemplateKind,
		APIVersion: metricsTemplateAPIVersion,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testMetricsTemplateNamespace,
		Name:      testMetricsTemplateName,
	},
}

var deployment = k8sapps.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testDeploymentNamespace,
		Name:      testDeploymentName,
	},
	Spec: k8sapps.DeploymentSpec{
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "hello-helidon",
				},
			},
		},
	},
}

// TestCreateJobName tests the job name creator
// GIVEN a set of names
// WHEN the function is invoked
// THEN verify the name is correctly given
func TestCreateJobName(t *testing.T) {
	assert := asserts.New(t)
	assert.Equal(fmt.Sprintf("%s_%s_%s", testMetricsBindingNamespace, testDeploymentName, deploymentKind), createJobName(&metricsBinding))
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
