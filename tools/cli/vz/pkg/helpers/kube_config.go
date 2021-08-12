// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"errors"
	"io/ioutil"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"
)

type TokenInfo struct {
	// Refresh token is a jwt token used to refresh the access token
	// +optional
	RefreshToken string `json:"refreshToken"`
	// Time until which access token will be alive
	// +optional
	AccessTokenExpTime int64 `json:"accessTokenExpTime"`
	// Time intil which refresh token will be alive
	// +optional
	RefreshTokenExpTime int64 `json:"refreshTokenExpTime"`
}

type NamedKeycloakTokenInfo struct {
	// Name is the nickname for this user
	Name string `json:"name"`
	// TokenInfo holds the token information
	TokenInfo TokenInfo `json:"tokenInfo"`
}

type Config struct {
	*clientcmdapi.Config `json:",inline"`
	KeycloakTokenInfos   []NamedKeycloakTokenInfo `json:"keycloakTokenInfo,omitempty"`
}

// Removes a context with given name from kubeconfig
func RemoveContextFromKubeConfig(name string) error {
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	pos := -1
	for i := 0; i < len(kubeConfig.Contexts); i++ {
		if kubeConfig.Contexts[i].Name == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig.Contexts = append(kubeConfig.Contexts[:pos], kubeConfig.Contexts[pos+1:]...)
	}
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Removes a cluster with given name from kubeconfig
func RemoveClusterFromKubeConfig(name string) error {
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	pos := -1
	for i := 0; i < len(kubeConfig.Clusters); i++ {
		if kubeConfig.Clusters[i].Name == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig.Clusters = append(kubeConfig.Clusters[:pos], kubeConfig.Clusters[pos+1:]...)
	}
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Removes a user with given name from kubeconfig
func RemoveUserFromKubeConfig(name string) error {
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	pos := -1
	for i := 0; i < len(kubeConfig.KeycloakTokenInfos); i++ {
		if kubeConfig.KeycloakTokenInfos[i].Name == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig.KeycloakTokenInfos = append(kubeConfig.KeycloakTokenInfos[:pos], kubeConfig.KeycloakTokenInfos[pos+1:]...)
	}
	pos = -1
	for i := 0; i < len(kubeConfig.AuthInfos); i++ {
		if kubeConfig.AuthInfos[i].Name == name {
			pos = i
			break
		}
	}
	if pos != -1 {
		kubeConfig.AuthInfos = append(kubeConfig.AuthInfos[:pos], kubeConfig.AuthInfos[pos+1:]...)
	}
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Changes current cluster to given cluster in kubeconfig
func SetCurrentContextInKubeConfig(name string) error {
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	kubeConfig.CurrentContext = name
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Adds a cluster to kubeconfig
func SetClusterInKubeConfig(name string, serverURL string, caData []byte) error {
	err := RemoveClusterFromKubeConfig(name)
	if err != nil {
		return err
	}
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	var currentCluster clientcmdapi.NamedCluster
	if len(caData) == 0 {
		currentCluster = clientcmdapi.NamedCluster{
			Name: name,
			Cluster: clientcmdapi.Cluster{
				Server: serverURL,
			},
		}
	} else {
		currentCluster = clientcmdapi.NamedCluster{
			Name: name,
			Cluster: clientcmdapi.Cluster{
				Server:                   serverURL,
				CertificateAuthorityData: caData,
			},
		}
	}
	kubeConfig.Clusters = append(kubeConfig.Clusters, currentCluster)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Adds a user to kubeconfig
func SetUserInKubeConfig(name string, accessToken string, authDetails AuthDetails) error {
	err := RemoveUserFromKubeConfig(name)
	if err != nil {
		return err
	}
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	currentUser := clientcmdapi.NamedAuthInfo{
		Name: name,
		AuthInfo: clientcmdapi.AuthInfo{
			Token: accessToken,
		},
	}
	kubeConfig.AuthInfos = append(kubeConfig.AuthInfos, currentUser)

	currentTokenInfo := NamedKeycloakTokenInfo{
		Name: name,
		TokenInfo: TokenInfo{
			RefreshToken:        authDetails.RefreshToken,
			AccessTokenExpTime:  authDetails.AccessTokenExpTime,
			RefreshTokenExpTime: authDetails.RefreshTokenExpTime,
		},
	}
	kubeConfig.KeycloakTokenInfos = append(kubeConfig.KeycloakTokenInfos, currentTokenInfo)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Adds a new context to kubeconfig
func SetContextInKubeConfig(name string, clusterName string, userName string) error {
	err := RemoveContextFromKubeConfig(name)
	if err != nil {
		return err
	}
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return err
	}
	currentContext := clientcmdapi.NamedContext{
		Name: name,
		Context: clientcmdapi.Context{
			AuthInfo: userName,
			Cluster:  clusterName,
		},
	}
	kubeConfig.Contexts = append(kubeConfig.Contexts, currentContext)
	err = WriteToKubeConfig(kubeConfig)
	return err
}

// Reads the kubeconfig into a interface map
func ReadKubeConfig() (Config, error) {
	// Obtain the default kubeconfig's location
	kubeConfig := Config{}
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return kubeConfig, err
	}

	byteSliceKubeconfig, err := ioutil.ReadFile(kubeConfigLoc)
	if err != nil {
		return kubeConfig, err
	}
	// Load the default kubeconfig's configuration into clientcmdapi object
	err = yaml.Unmarshal(byteSliceKubeconfig, &kubeConfig)
	if err != nil {
		return kubeConfig, err
	}
	return kubeConfig, nil
}

// Writes the given interface map to kubeconfig
func WriteToKubeConfig(kubeConfig Config) error {
	// Write the new configuration into the default kubeconfig file
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	byteSliceKubeConfig, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(kubeConfigLoc, byteSliceKubeConfig, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Returns the current context in kubeconfig
func GetCurrentContextFromKubeConfig() (string, error) {
	var currentContext string
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return currentContext, err
	}
	currentContext = kubeConfig.CurrentContext
	return currentContext, nil
}

// Struct to store user's authentication data
type AuthDetails struct {
	RefreshTokenExpTime int64
	AccessTokenExpTime  int64
	RefreshToken        string
}

// Returns tokens and expiration times wrapped up in a struct
func GetAuthDetails(name string) (AuthDetails, error) {
	var authDetails AuthDetails
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return authDetails, err
	}

	pos := -1
	for i := 0; i < len(kubeConfig.KeycloakTokenInfos); i++ {
		if kubeConfig.KeycloakTokenInfos[i].Name == name {
			pos = i
			break
		}
	}
	if pos == -1 {
		return authDetails, errors.New("No user with nickname verrazzano")
	}

	refreshToken := kubeConfig.KeycloakTokenInfos[pos].TokenInfo.RefreshToken
	if len(refreshToken) == 0 {
		return authDetails, errors.New("Refresh Token not found in kubeconfig")
	}
	accesTokenExpTime := kubeConfig.KeycloakTokenInfos[pos].TokenInfo.AccessTokenExpTime
	if accesTokenExpTime == 0 {
		return authDetails, errors.New("Access Token Expiration Time not found in kubeconfig")
	}
	refreshTokenExpTime := kubeConfig.KeycloakTokenInfos[pos].TokenInfo.RefreshTokenExpTime
	if refreshTokenExpTime == 0 {
		return authDetails, errors.New("Refresh Token Expiration Time not found in kubeconfig")
	}
	authDetails.RefreshToken = refreshToken
	authDetails.AccessTokenExpTime = accesTokenExpTime
	authDetails.RefreshTokenExpTime = refreshTokenExpTime
	return authDetails, nil
}

// Returns the certificate authority data already present in kubeconfig
func GetCAData(name string) ([]byte, error) {
	var caData []byte
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return caData, err
	}
	pos := -1
	for i := 0; i < len(kubeConfig.Clusters); i++ {
		if kubeConfig.Clusters[i].Name == name {
			pos = i
			break
		}
	}
	if pos == -1 {
		return caData, errors.New("Unable to find cluster with nickname verrazzano")
	}
	// If caData is not present, it will return a empty string
	caData = kubeConfig.Clusters[pos].Cluster.CertificateAuthorityData
	return caData, nil
}

// Returns the verrazzano api server url
func GetVerrazzanoAPIURL(name string) (string, error) {
	var verrazzanoAPIURL string
	kubeConfig, err := ReadKubeConfig()
	if err != nil {
		return verrazzanoAPIURL, err
	}
	pos := -1
	for i := 0; i < len(kubeConfig.Clusters); i++ {
		if kubeConfig.Clusters[i].Name == name {
			pos = i
			break
		}
	}
	if pos == -1 {
		return verrazzanoAPIURL, errors.New("Unable to find cluster with nickname verrazzano")
	}
	// If caData is not present, it will return a empty string
	verrazzanoAPIURL = kubeConfig.Clusters[pos].Cluster.Server
	return verrazzanoAPIURL, nil
}
