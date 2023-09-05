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
	configName         = "securityconfig-secret"
	configNamespace    = "verrazzano-logging"
	securityConfigYaml = "/verrazzano/platform-operator/thirdparty/manifests/opensearch-securityconfig.yaml"
)

// MergeSecurityConfigs merges a security config secret yamls and the helm
// func MergeSecurityConfigs(actualCR *corev1.Secret, helmSecretYaml []byte) ([]byte, error) {
func MergeSecretData(ctx spi.ComponentContext) error {
	// Read the contents of the first YAML file into a byte slice.
	helmYaml, err := os.ReadFile(securityConfigYaml)
	if err != nil {
		return err
	}
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}

	secret, err := clientset.CoreV1().Secrets(configNamespace).Get(context.TODO(), configName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error")
		return err

	}
	// Unmarshal the first YAML data into a map[string]interface{}.
	dataHelmSecret := make(map[string]interface{})
	err = yaml.Unmarshal(helmYaml, &dataHelmSecret)
	if err != nil {
		fmt.Println("Error")
		return err
	}

	// Get the YAML data from Helm Secret and Secret
	configYamlHelm := getYamlData(dataHelmSecret, "config.yml")
	configYamlSecret := getSecretData(secret, "config.yml")

	// Unmarshal the YAML data into maps
	dataHelmConfig, err := unmarshalYAML(configYamlHelm)
	if err != nil {
		fmt.Println("Error")
		return err
	}
	dataSecretConfig, err := unmarshalYAML(configYamlSecret)
	if err != nil {
		fmt.Println("Error")
		return err
	}
	// Merge the config.yml data
	mergedConfig, err := mergeConfigData(dataSecretConfig, dataHelmConfig)
	if err != nil {
		fmt.Println("Error")
		return err
	}
	// Convert the merged data back to YAML format
	mergedConfigYAML, err := yaml.Marshal(mergedConfig)
	if err != nil {
		fmt.Println("Error")
		return err
	}

	// Update the secret with the merged config data
	secret.Data["config.yml"] = mergedConfigYAML

	// Get the YAML data from Helm Secret and Secret
	usersYamlHelm := getYamlData(dataHelmSecret, "internal_users.yml")
	usersYamlSecret := getSecretData(secret, "internal_users.yml")

	// Unmarshal the YAML data into maps
	dataHelmUsers, err := unmarshalYAML(usersYamlHelm)
	if err != nil {
		return err
	}
	dataSecretUsers, err := unmarshalYAML(usersYamlSecret)
	if err != nil {
		return err
	}
	adminSecret, err := clientset.CoreV1().Secrets(configNamespace).Get(context.TODO(), "admin-credentials-secret", metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error")
		return err
	}
	adminHash := getAdminHash(adminSecret)
	// Merge the config.yml data
	mergedUsers, err := mergeUserData(dataSecretUsers, dataHelmUsers, adminHash)
	if err != nil {
		return err
	}

	// Convert the merged data back to YAML format
	mergedUsersYAML, err := yaml.Marshal(mergedUsers)
	if err != nil {
		return err
	}

	// Assign the YAML byte slice to the secret data
	secret.Data["config.yml"] = mergedUsersYAML

	clientset.CoreV1().Secrets(configNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})

	return nil
}

func getAdminHash(secret *corev1.Secret) string {
	hashBytes, ok := secret.Data["hash"]
	if !ok {
		// Handle the case where the 'hash' key is not present in the secret
		return "" // or return an error, depending on your use case
	}
	return string(hashBytes)
}

func getSecretData(secret *corev1.Secret, yamlName string) []byte {
	var byteYaml []byte
	for key, val := range secret.Data {
		if key == yamlName {
			byteYaml = val

		}
	}
	return byteYaml
}
func unmarshalYAML(yamlData []byte) (map[interface{}]interface{}, error) {
	var data map[interface{}]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func getYamlData(helmMap map[string]interface{}, yamlName string) []byte {
	stringData := helmMap["stringData"].(map[interface{}]interface{})[yamlName].(string)
	return []byte(stringData)
}

func mergeConfigData(data1, data2 map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	mergedData := make(map[string]interface{})
	values1, ok1 := data1["config"].(map[interface{}]interface{})["dynamic"].(map[interface{}]interface{})["authc"].(map[interface{}]interface{})
	values2, ok2 := data2["config"].(map[interface{}]interface{})["dynamic"].(map[interface{}]interface{})["authc"].(map[interface{}]interface{})
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("config data structure mismatch")
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
func mergeUserData(data1, data2 map[interface{}]interface{}, hashFromSecret string) (map[interface{}]interface{}, error) {
	mergedData := make(map[interface{}]interface{})
	// Update the 'hash' field in mergedData1
	adminData, ok := data1["admin"].(map[interface{}]interface{})
	if !ok {
		fmt.Println("Error")
		return nil, fmt.Errorf("user data structure mismatch")
	}
	adminData["hash"] = hashFromSecret
	for key1, val1 := range data1 {
		fmt.Println(key1)
		if key1 == "admin" {
			mergedData[key1.(string)] = adminData
		} else {
			mergedData[key1.(string)] = val1
		}
	}
	for key2, val2 := range data2 {
		if _, ok := mergedData[key2.(string)]; ok {
			continue
		} else {
			mergedData[key2.(string)] = val2
		}
	}
	return mergedData, nil
}
