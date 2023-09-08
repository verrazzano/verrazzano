// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	securitySecretName = "securityconfig-secret"
	securityNamespace  = "verrazzano-logging"
	securityConfigYaml = "../../../../thirdparty/manifests/opensearch-operator/opensearch-securityconfig.yaml"
	configYaml         = "config.yml"
	usersYaml          = "internal_users.yml"
	adminName          = "admin-credentials-secret"
)

// getClient returns a controller runtime client for the Verrazzano resource
func getClient(ctx spi.ComponentContext) (client.Client, error) {
	return ctx.Client(), nil
}

// MergeSecretData merges a security config secret
func MergeSecretData(ctx spi.ComponentContext) error {
	securityYaml, err := os.ReadFile(securityConfigYaml)
	if err != nil {
		return err
	}
	securityYamlData := make(map[string]interface{})
	err = yaml.Unmarshal(securityYaml, &securityYamlData)
	if err != nil {
		return err
	}
	configYamlFile, err := getYamlData(securityYamlData, configYaml)
	if err != nil {
		return err
	}
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	var scr corev1.Secret
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, &scr); err != nil {
		return err
	}

	configYamlSecret, err := getSecretData(&scr, configYaml)
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
	usersYamlSecret, err := getSecretData(&scr, usersYaml)
	if err != nil {
		return err
	}

	dataFileUsers, err := unmarshalYAML(usersYamlFile)
	if err != nil {
		return err
	}
	dataSecretUsers, err := unmarshalYAML(usersYamlSecret)
	if err != nil {
		return err
	}
	var adminSecret corev1.Secret
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: adminName}, &adminSecret); err != nil {
		return err
	}

	adminHash, err := getAdminHash(&adminSecret)
	if err != nil {
		return err
	}
	// Merge the internal_users.yml data
	mergedUsers, err := mergeUserYamlData(dataFileUsers, dataSecretUsers, adminHash)
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
	if err := client.Update(context.TODO(), &scr); err != nil {
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
func unmarshalYAML(yamlData []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func getYamlData(helmMap map[string]interface{}, yamlName string) ([]byte, error) {
	stringData, ok := helmMap["stringData"].(map[string]interface{})[yamlName].(string)
	if !ok {
		return nil, fmt.Errorf("error finding config data")
	}
	return []byte(stringData), nil
}

func mergeConfigYamlData(data1, data2 map[string]interface{}) (map[string]interface{}, error) {
	mergedData := make(map[string]interface{})
	values1, ok1 := data1["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"].(map[string]interface{})
	values2, ok2 := data2["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"].(map[string]interface{})
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("config not found")
	}
	for key1, val1 := range values1 {
		mergedData[key1] = val1
	}
	for key2, val2 := range values2 {
		if _, ok := mergedData[key2]; ok && strings.HasPrefix(key2, "vz") {
			continue
		}
		mergedData[key2] = val2
	}
	data1["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"] = mergedData
	return data1, nil
}
func mergeUserYamlData(data1, data2 map[string]interface{}, hashFromSecret string) (map[string]interface{}, error) {
	mergedData := make(map[string]interface{})
	adminData, ok := data1["admin"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	adminData["hash"] = hashFromSecret
	for key1, val1 := range data1 {
		if key1 == "admin" {
			mergedData[key1] = adminData
		} else {
			mergedData[key1] = val1
		}
	}
	for key2, val2 := range data2 {
		if _, exists := mergedData[key2]; exists {
			continue
		} else {
			mergedData[key2] = val2
		}
	}
	return mergedData, nil
}
