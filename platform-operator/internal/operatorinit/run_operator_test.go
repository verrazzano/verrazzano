// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"testing"
)

// TestGenerateConfigMapFromHelmChartFiles tests the call to generateConfigMapFromHelmChartFiles
// GIVEN the chart directory of the verrazzano-platform-operator
//
//	WHEN I call generateConfigMapFromHelmChartFiles
//	THEN no error is returned and a config map is generated with expected results
func TestGenerateConfigMapFromHelmChartFiles(t *testing.T) {
	vpoHelmChartConfigMap := generateVPOConfigMap(t)
	assert.Equal(t, vpoHelmChartConfigMapName, vpoHelmChartConfigMap.Name)
	assert.Equal(t, constants.VerrazzanoInstallNamespace, vpoHelmChartConfigMap.Namespace)
	assert.Equal(t, 15, len(vpoHelmChartConfigMap.Data))
	assert.Contains(t, vpoHelmChartConfigMap.Data, "crds...install.verrazzano.io_verrazzanos.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...clusterrole.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...clusterrolebinding.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...deployment.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...install.verrazzano.io_modules.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...meta-configmap.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...mutatingWebHookConfiguration.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...namespace.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...service.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...serviceaccount.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "templates...validatingwebhookconfiguration.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, ".helmignore")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "Chart.yaml")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "NOTES.txt")
	assert.Contains(t, vpoHelmChartConfigMap.Data, "values.yaml")
}

// TestCreateVPOHelmChartConfigMap tests the call to createVPOHelmChartConfigMap
// GIVEN a config map with a helm chart
//
//	WHEN I call createVPOHelmChartConfigMap
//	THEN no error is returned and a config map is created/updated with expected results
func TestCreateVPOHelmChartConfigMap(t *testing.T) {
	kubeclient := fake.NewSimpleClientset()

	// Test the create of the VPO config map
	configMap := generateVPOConfigMap(t)
	err := createVPOHelmChartConfigMap(kubeclient, configMap)
	assert.NoError(t, err)
	create, err := kubeclient.CoreV1().ConfigMaps(configMap.Namespace).Get(context.TODO(), vpoHelmChartConfigMapName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Contains(t, create.Data, "crds...install.verrazzano.io_verrazzanos.yaml")

	// Test the update of the VPO config map by using the VMO helm chart
	configMap = generateVPOConfigMapWithVMOChart(t)
	err = createVPOHelmChartConfigMap(kubeclient, configMap)
	assert.NoError(t, err)
	update, err := kubeclient.CoreV1().ConfigMaps(configMap.Namespace).Get(context.TODO(), vpoHelmChartConfigMapName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Contains(t, update.Data, "crds...verrazzano.io_verrazzanomonitoringinstances_crd.yaml")
}

func generateVPOConfigMap(t *testing.T) *corev1.ConfigMap {
	config.TestHelmConfigDir = "../../helm_config"
	chartDir := config.GetHelmVPOChartsDir()
	files, err := os.ReadDir(chartDir)
	assert.NoError(t, err)
	vpoHelmChartConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpoHelmChartConfigMapName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
	}

	err = generateConfigMapFromHelmChartFiles(chartDir, "", files, vpoHelmChartConfigMap)
	assert.NoError(t, err)

	return vpoHelmChartConfigMap
}

func generateVPOConfigMapWithVMOChart(t *testing.T) *corev1.ConfigMap {
	config.TestHelmConfigDir = "../../helm_config"
	chartDir := config.GetHelmVMOChartsDir()
	files, err := os.ReadDir(chartDir)
	assert.NoError(t, err)
	vpoHelmChartConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpoHelmChartConfigMapName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
	}

	err = generateConfigMapFromHelmChartFiles(chartDir, "", files, vpoHelmChartConfigMap)
	assert.NoError(t, err)

	return vpoHelmChartConfigMap
}
