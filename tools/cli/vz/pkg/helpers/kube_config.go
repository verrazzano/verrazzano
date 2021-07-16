// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"encoding/base64"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() string {

	var kubeConfig string
	kubeConfigEnvVar := os.Getenv("KUBECONFIG")

	if len(kubeConfigEnvVar) > 0 {
		// Find using environment variables
		kubeConfig = kubeConfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		kubeConfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		panic("Unable to find kubeconfig")
	}
	return kubeConfig
}

func RemoveContextFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	pos := -1
	for i := 0; i < len(kubeConfig["contexts"].([]interface{})); i++ {
		if kubeConfig["contexts"].([]interface{})[i].(map[string]interface{})["name"] == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig["contexts"] = append(kubeConfig["contexts"].([]interface{})[:pos], kubeConfig["contexts"].([]interface{})[pos+1:]...)
	}
	WriteToKubeConfig(kubeConfig)
}

func RemoveClusterFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	pos := -1
	for i := 0; i < len(kubeConfig["clusters"].([]interface{})); i++ {
		if kubeConfig["clusters"].([]interface{})[i].(map[string]interface{})["name"] == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig["clusters"] = append(kubeConfig["clusters"].([]interface{})[:pos], kubeConfig["clusters"].([]interface{})[pos+1:]...)
	}
	WriteToKubeConfig(kubeConfig)
}

func RemoveUserFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	pos := -1
	for i := 0; i < len(kubeConfig["users"].([]interface{})); i++ {
		if kubeConfig["users"].([]interface{})[i].(map[string]interface{})["name"] == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig["users"] = append(kubeConfig["users"].([]interface{})[:pos], kubeConfig["users"].([]interface{})[pos+1:]...)
	}
	WriteToKubeConfig(kubeConfig)
}

func SetCurrentContextInKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	kubeConfig["current-context"] = name
	WriteToKubeConfig(kubeConfig)
}

func SetClusterInKubeConfig(name string, serverURL string, caData []byte) {
	RemoveClusterFromKubeConfig(name)
	kubeConfig := ReadKubeConfig()
	currentCluster := make(map[string]interface{})
	currentCluster["name"] = name
	currentClusterInfo := make(map[string]interface{})
	currentClusterInfo["server"] = serverURL
	currentClusterInfo["certificate-authority-data"] = base64.StdEncoding.EncodeToString(caData)
	currentCluster["cluster"] = currentClusterInfo
	kubeConfig["clusters"] = append(kubeConfig["clusters"].([]interface{}), currentCluster)
	WriteToKubeConfig(kubeConfig)
}

func SetUserInKubeConfig(name string, accessToken string, refreshToken string, accessTokenExpTime int64, refreshTokenExpTime int64) {
	RemoveUserFromKubeConfig(name)
	kubeConfig := ReadKubeConfig()
	currentUser := make(map[string]interface{})
	currentUser["name"] = name
	currentUserInfo := make(map[string]interface{})
	currentUserInfo["token"] = accessToken
	currentUserInfo["refreshToken"] = refreshToken
	currentUserInfo["accessTokenExpTime"] = accessTokenExpTime
	currentUserInfo["refreshTokenExpTime"] = refreshTokenExpTime
	currentUser["user"] = currentUserInfo
	kubeConfig["users"] = append(kubeConfig["users"].([]interface{}), currentUser)
	WriteToKubeConfig(kubeConfig)
}

func SetContextInKubeConfig(name string, clusterName string, userName string) {
	RemoveContextFromKubeConfig(name)
	kubeConfig := ReadKubeConfig()
	currentContext := make(map[string]interface{})
	currentContext["name"] = name
	currentContextInfo := make(map[string]interface{})
	currentContextInfo["cluster"] = clusterName
	currentContextInfo["user"] = userName
	currentContext["context"] = currentContextInfo
	kubeConfig["contexts"] = append(kubeConfig["contexts"].([]interface{}), currentContext)
	WriteToKubeConfig(kubeConfig)
}

func ReadKubeConfig() map[string]interface{} {
	// Obtain the default kubeconfig's location
	kubeConfigLoc := GetKubeConfigLocation()
	byteSliceKubeconfig, err := ioutil.ReadFile(kubeConfigLoc)
	if err != nil {
		panic("Unable to read kubeconfig")
	}
	kubeConfig := make(map[string]interface{})
	// Load the default kubeconfig's configuration into clientcmdapi object
	err = yaml.Unmarshal(byteSliceKubeconfig, &kubeConfig)
	if err != nil {
		panic("Unable to unmarshall kubeconfig")
	}
	return kubeConfig
}

func WriteToKubeConfig(kubeConfig map[string]interface{}) {
	// Write the new configuration into the default kubeconfig file
	kubeConfigLoc := GetKubeConfigLocation()
	byteSliceKubeConfig, err := yaml.Marshal(kubeConfig)
	if err != nil {
		panic("Unable to marshall kubeconfig")
	}
	err = ioutil.WriteFile(kubeConfigLoc, byteSliceKubeConfig, 0644)
	if err != nil {
		panic("Unable to write to kubeconfig")
	}
}

func GetCurrentContextFromKubeConfig() string {
	kubeConfig := ReadKubeConfig()
	return kubeConfig["current-context"].(string)
}

func GetAuthDetails() (int64, int64, string, string) {
	kubeConfig := ReadKubeConfig()
	pos := -1
	for i := 0; i < len(kubeConfig["users"].([]interface{})); i++ {
		if kubeConfig["users"].([]interface{})[i].(map[string]interface{})["name"] == "verrazzano" {
			pos = i
			break
		}
	}
	if pos == -1 {
		panic("No user with nickname verrazzano")
	}
	currentUserInfo := kubeConfig["users"].([]interface{})[pos].(map[string]interface{})["user"].(map[string]interface{})
	return int64(currentUserInfo["accessTokenExpTime"].(float64)), int64(currentUserInfo["refreshTokenExpTime"].(float64)), currentUserInfo["token"].(string), currentUserInfo["refreshToken"].(string)
}

func GetCAData() string {
	kubeConfig := ReadKubeConfig()
	pos := -1
	for i := 0; i < len(kubeConfig["clusters"].([]interface{})); i++ {
		if kubeConfig["clusters"].([]interface{})[i].(map[string]interface{})["name"] == "verrazzano" {
			pos = i
			break
		}
	}
	if pos == -1 {
		panic("No user with nickname verrazzano")
	}
	currentUserInfo := kubeConfig["clusters"].([]interface{})[pos].(map[string]interface{})["cluster"].(map[string]interface{})
	return currentUserInfo["certificate-authority-data"].(string)
}
