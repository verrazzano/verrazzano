// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	securitySecretName = "securityconfig-secret"
	securityNamespace  = "verrazzano-logging"
	securityConfigYaml = "/verrazzano/platform-operator/thirdparty/manifests/opensearch-securityconfig.yaml"
	configYaml         = "config.yml"
	usersYaml          = "internals_users.yml"
	adminName          = "admin-credentials-secret"
)

// MergeSecretData merges a security config secret
func MergeSecretData(ctx spi.ComponentContext) error {
	securityYaml, err := os.ReadFile(securityConfigYaml)
	if err != nil {
		return err
	}
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	scr, err := clientset.CoreV1().Secrets(securityNamespace).Get(context.TODO(), securitySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	securityYamlData := make(map[string]interface{})
	err = yaml.Unmarshal(securityYaml, &securityYamlData)
	if err != nil {
		return err
	}
	// Get the YAML data from Helm Secret and Secret
	configYamlFile, err := getYamlData(securityYamlData, configYaml)
	if err != nil {
		return err
	}
	configYamlSecret, err := getSecretData(scr, configYaml)
	if err != nil {
		return err
	}

	// Unmarshal the YAML data into maps
	dataFileConfig, err := unmarshalYAML(configYamlFile)
	if err != nil {
		return err
	}
	dataSecretConfig, err := unmarshalYAML(configYamlSecret)
	if err != nil {
		return err
	}
	mergedConfig, err := mergeConfigYamlData(dataSecretConfig, dataFileConfig)
	if err != nil {
		return err
	}
	mergedConfigYAML, err := yaml.Marshal(mergedConfig)
	if err != nil {
		return err
	}

	// Update the secret with the merged config data
	scr.Data[configYaml] = mergedConfigYAML

	usersYamlFile, err := getYamlData(securityYamlData, usersYaml)
	if err != nil {
		return err
	}
	usersYamlSecret, err := getSecretData(scr, usersYaml)
	if err != nil {
		return err
	}

	dataHelmUsers, err := unmarshalYAML(usersYamlFile)
	if err != nil {
		return err
	}
	dataSecretUsers, err := unmarshalYAML(usersYamlSecret)
	if err != nil {
		return err
	}
	adminSecret, err := clientset.CoreV1().Secrets(securityNamespace).Get(context.TODO(), adminName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	adminHash, err := getAdminHash(adminSecret)
	if err != nil {
		return err
	}
	// Merge the internal_users.yml data
	mergedUsers, err := mergeUserYamlData(dataHelmUsers, dataSecretUsers, adminHash)
	if err != nil {
		return err
	}
	mergedUsersYAML, err := yaml.Marshal(mergedUsers)
	if err != nil {
		return err
	}
	// Assign the YAML byte slice to the secret data
	scr.Data[usersYaml] = mergedUsersYAML

	// Update the secret
	_, err = clientset.CoreV1().Secrets(securityNamespace).Update(context.TODO(), scr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func getAdminHash(secret *corev1.Secret) (string, error) {
	hashBytes, ok := secret.Data["hash"]
	if !ok {
		// Handle the case where the 'hash' key is not present in the secret
		return "", fmt.Errorf("hash not found")
	}
	return string(hashBytes), nil
}

func getSecretData(secret *corev1.Secret, yamlName string) ([]byte, error) {
	var byteYaml []byte
	for key, val := range secret.Data {
		if key == yamlName {
			byteYaml = val

		}
	}
	return byteYaml, nil
}
func unmarshalYAML(yamlData []byte) (map[interface{}]interface{}, error) {
	var data map[interface{}]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func getYamlData(helmMap map[string]interface{}, yamlName string) ([]byte, error) {
	stringData, ok := helmMap["stringData"].(map[interface{}]interface{})[yamlName].(string)
	if !ok {
		return nil, fmt.Errorf("error finding config data")
	}
	return []byte(stringData), nil
}

func mergeConfigYamlData(data1, data2 map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	mergedData := make(map[string]interface{})
	values1, ok1 := data1["config"].(map[interface{}]interface{})["dynamic"].(map[interface{}]interface{})["authc"].(map[interface{}]interface{})
	values2, ok2 := data2["config"].(map[interface{}]interface{})["dynamic"].(map[interface{}]interface{})["authc"].(map[interface{}]interface{})
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("config not found")
	}
	for key1, val1 := range values1 {
		mergedData[key1.(string)] = val1
	}
	for key2, val2 := range values2 {
		if _, ok := mergedData[key2.(string)]; ok && strings.HasPrefix(key2.(string), "vz") {
			continue
		}
		mergedData[key2.(string)] = val2
	}
	data1["config"].(map[interface{}]interface{})["dynamic"].(map[interface{}]interface{})["authc"] = mergedData
	return data1, nil
}
func mergeUserYamlData(data1, data2 map[interface{}]interface{}, hashFromSecret string) (map[interface{}]interface{}, error) {
	mergedData := make(map[interface{}]interface{})
	adminData, ok := data1["admin"].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("user data structure mismatch")
	}
	adminData["hash"] = hashFromSecret
	for key1, val1 := range data1 {
		if key1 == "admin" {
			mergedData[key1.(string)] = adminData
		} else {
			mergedData[key1.(string)] = val1
		}
	}
	for key2, val2 := range data2 {
		if _, exists := mergedData[key2.(string)]; exists {
			continue
		} else {
			mergedData[key2.(string)] = val2
		}
	}
	return mergedData, nil
}
