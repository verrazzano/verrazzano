// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() (string,error) {

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
		return kubeConfig,errors.New("Unable to find kubeconfig")
	}
	return kubeConfig,nil
}

func RemoveContextFromKubeConfig(name string) error {
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
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
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func RemoveClusterFromKubeConfig(name string) error {
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
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
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func RemoveUserFromKubeConfig(name string) error {
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
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
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func SetCurrentContextInKubeConfig(name string) error {
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
	kubeConfig["current-context"] = name
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func SetClusterInKubeConfig(name string, serverURL string, caData []byte) error {
	err := RemoveClusterFromKubeConfig(name)
	if err!=nil {
		return err
	}
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
	currentCluster := make(map[string]interface{})
	currentCluster["name"] = name
	currentClusterInfo := make(map[string]interface{})
	currentClusterInfo["server"] = serverURL
	currentClusterInfo["certificate-authority-data"] = base64.StdEncoding.EncodeToString(caData)
	currentCluster["cluster"] = currentClusterInfo
	kubeConfig["clusters"] = append(kubeConfig["clusters"].([]interface{}), currentCluster)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func SetUserInKubeConfig(name string, authDetails AuthDetails) error {
	err := RemoveUserFromKubeConfig(name)
	if err!=nil {
		return err
	}
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
	currentUser := make(map[string]interface{})
	currentUser["name"] = name
	currentUserInfo := make(map[string]interface{})
	currentUserInfo["token"] = authDetails.AccessToken
	currentUserInfo["refreshToken"] = authDetails.RefreshToken
	currentUserInfo["accessTokenExpTime"] = authDetails.AccessTokenExpTime
	currentUserInfo["refreshTokenExpTime"] = authDetails.RefreshTokenExpTime
	currentUser["user"] = currentUserInfo
	kubeConfig["users"] = append(kubeConfig["users"].([]interface{}), currentUser)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func SetContextInKubeConfig(name string, clusterName string, userName string) error {
	err := RemoveContextFromKubeConfig(name)
	if err!=nil {
		return err
	}
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return err
	}
	currentContext := make(map[string]interface{})
	currentContext["name"] = name
	currentContextInfo := make(map[string]interface{})
	currentContextInfo["cluster"] = clusterName
	currentContextInfo["user"] = userName
	currentContext["context"] = currentContextInfo
	kubeConfig["contexts"] = append(kubeConfig["contexts"].([]interface{}), currentContext)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

func ReadKubeConfig() (map[string]interface{},error) {
	// Obtain the default kubeconfig's location
	var kubeConfig map[string]interface{}
	kubeConfigLoc,err := GetKubeConfigLocation()
	if err!=nil {
		return kubeConfig, err
	}
	byteSliceKubeconfig, err := ioutil.ReadFile(kubeConfigLoc)
	if err != nil {
		return kubeConfig, err
	}
	kubeConfig = make(map[string]interface{})
	// Load the default kubeconfig's configuration into clientcmdapi object
	err = yaml.Unmarshal(byteSliceKubeconfig, &kubeConfig)
	if err != nil {
		return kubeConfig, err
	}
	return kubeConfig, nil
}

func WriteToKubeConfig(kubeConfig map[string]interface{}) error {
	// Write the new configuration into the default kubeconfig file
	kubeConfigLoc,err := GetKubeConfigLocation()
	if err!=nil {
		return err
	}
	byteSliceKubeConfig, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(kubeConfigLoc, byteSliceKubeConfig, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetCurrentContextFromKubeConfig() (string,error) {
	var currentContext string
	kubeConfig, err := ReadKubeConfig()
	if err!=nil {
		return currentContext, err
	}
	currentContext = kubeConfig["current-context"].(string)
	return currentContext, nil
}

type AuthDetails struct{
	AccessTokenExpTime int64
	RefreshTokenExpTime int64
	AccessToken string
	RefreshToken string
}

// Returns tokens and expiration times wrapped up in a struct
func GetAuthDetails() (AuthDetails,error) {
	var authDetails AuthDetails
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return authDetails, err
	}
	pos := -1
	for i := 0; i < len(kubeConfig["users"].([]interface{})); i++ {
		if kubeConfig["users"].([]interface{})[i].(map[string]interface{})["name"] == Verrazzano {
			pos = i
			break
		}
	}
	if pos == -1 {
		return authDetails,errors.New("No user with nickname verrazzano")
	}
	currentUserInfo := kubeConfig["users"].([]interface{})[pos].(map[string]interface{})["user"].(map[string]interface{})
	accesToken,ok := currentUserInfo["token"]
	if !ok {
		return authDetails,errors.New("Access Token not found in kubeconfig")
	}
	refreshToken, ok := currentUserInfo["refreshToken"]
	if !ok {
		return authDetails,errors.New("Refresh Token not found in kubeconfig")
	}
	accesTokenExpTime,ok := currentUserInfo["accessTokenExpTime"]
	if !ok {
		return authDetails,errors.New("Access Token Expiration Time not found in kubeconfig")
	}
	refreshTokenExpTime, ok := currentUserInfo["refreshTokenExpTime"]
	if !ok{
		return authDetails,errors.New("Refresh Token Expiration Time not found in kubeconfig")
	}
	authDetails.AccessToken = accesToken.(string)
	authDetails.RefreshToken = refreshToken.(string)
	authDetails.AccessTokenExpTime = int64(accesTokenExpTime.(float64))
	authDetails.RefreshTokenExpTime = int64(refreshTokenExpTime.(float64))
	return authDetails, nil
}

// Returns the certificate authority data already present in kubeconfig
func GetCAData() (string,error) {
	var caData string
	kubeConfig,err := ReadKubeConfig()
	if err!=nil {
		return caData,err
	}
	pos := -1
	for i := 0; i < len(kubeConfig["clusters"].([]interface{})); i++ {
		if kubeConfig["clusters"].([]interface{})[i].(map[string]interface{})["name"] == Verrazzano {
			pos = i
			break
		}
	}
	if pos == -1 {
		return caData,errors.New("Unable to find cluster with nick name verrazzano")
	}
	currentUserInfo := kubeConfig["clusters"].([]interface{})[pos].(map[string]interface{})["cluster"].(map[string]interface{})
	caData = currentUserInfo["certificate-authority-data"].(string)
	return caData, nil
}
