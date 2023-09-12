// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"path"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	securitySecretName = "securityconfig-secret"
	securityNamespace  = constants.VerrazzanoLoggingNamespace
	securityConfigYaml = "opensearch-operator/opensearch-securityconfig.yaml"
	configYaml         = "config.yml"
	usersYaml          = "internal_users.yml"
	hashSecName        = "admin-credentials-secret"
)

// getClient returns a controller runtime client for the Verrazzano resource
func getClient(ctx spi.ComponentContext) (client.Client, error) {
	return ctx.Client(), nil
}

// MergeSecretData merges config.yml and internal_users.yml from the security config secret and the
// helm config present in manifests directory.
func MergeSecretData(ctx spi.ComponentContext, helmManifestsDir string) error {
	// Get the security config yaml from the manifests directory and
	//extract the config.yml and internals_users.yml data
	filePath := path.Join(helmManifestsDir, securityConfigYaml)
	securityYaml, err := os.ReadFile(filePath)
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

	// Get the security config secret and
	// extract the config.yml and internals_users.yml data
	var scr corev1.Secret
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: securitySecretName}, &scr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Do nothing and return if secret doesn't exist
			return nil
		}
		return err
	}
	configYamlSecret, err := getSecretYamlData(&scr, configYaml)
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
	usersYamlSecret, err := getSecretYamlData(&scr, usersYaml)
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
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: securityNamespace, Name: hashSecName}, &adminSecret); err != nil {
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

// getAdminHash fetches the hash value from the admin secret
func getAdminHash(secret *corev1.Secret) (string, error) {
	hashBytes, ok := secret.Data["hash"]
	if !ok {
		// Handle the case where the 'hash' key is not present in the secret
		return "", fmt.Errorf("hash not found")
	}
	return string(hashBytes), nil
}

// getSecretYamlData gets the data corresponding to the yaml name
func getSecretYamlData(secret *corev1.Secret, yamlName string) ([]byte, error) {
	var byteYaml []byte
	for key, val := range secret.Data {
		if key == yamlName {
			byteYaml = val

		}
	}
	return byteYaml, nil
}

// unmarshalYAML unmarshals the data into the map
func unmarshalYAML(yamlData []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// getYamlData gets the given data from the helm config yaml
func getYamlData(helmMap map[string]interface{}, yamlName string) ([]byte, error) {
	// Check if "stringData" exists and is of the expected type
	stringData, ok := helmMap["stringData"]
	if !ok {
		return nil, fmt.Errorf("stringData is not found in the yaml")
	}

	// Assert the type of stringData to map[interface{}]interface{}
	stringDataMap, ok := stringData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("stringData is not of type map[string]interface{}")
	}

	// Check if "yamlName" exists in stringDataMap
	dataValue, ok := stringDataMap[yamlName]
	if !ok {
		return nil, fmt.Errorf("%s not found in stringData", yamlName)
	}

	// Convert dataValue to a string
	stringValue, ok := dataValue.(string)
	if !ok {
		return nil, fmt.Errorf("%s is not a string", yamlName)
	}
	return []byte(stringValue), nil
}

// mergeConfigYamlData merges the config.yml data from the secret and the helm config
func mergeConfigYamlData(dataSecret, dataFile map[string]interface{}) (map[string]interface{}, error) {
	mergedData := make(map[string]interface{})
	authcFile, ok := dataFile["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config not found")
	}
	authcSecret := make(map[string]interface{})
	configSecret, ok := dataSecret["config"].(map[string]interface{})
	if ok {
		dynamicSecret, ok := configSecret["dynamic"].(map[string]interface{})
		if ok {
			authcSecret = dynamicSecret["authc"].(map[string]interface{})
		}
	}
	for key1, val1 := range authcFile {
		mergedData[key1] = val1
	}
	for key2, val2 := range authcSecret {
		if _, ok := mergedData[key2]; ok && strings.HasPrefix(key2, "vz") {
			continue
		}
		mergedData[key2] = val2
	}
	dataSecret["config"].(map[string]interface{})["dynamic"].(map[string]interface{})["authc"] = mergedData
	return dataSecret, nil
}

// mergeUserYamlData merges the internal_users.yml data from the secret and the helm config
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
