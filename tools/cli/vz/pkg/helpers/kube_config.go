// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
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

func RemoveClusterFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	delete(kubeConfig.Clusters, name)
	WriteToKubeConfig(kubeConfig)
}

func RemoveUserFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	delete(kubeConfig.AuthInfos, name)
	WriteToKubeConfig(kubeConfig)
}

func RemoveContextFromKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	delete(kubeConfig.Contexts, name)
	WriteToKubeConfig(kubeConfig)
}

func SetCurrentContextInKubeConfig(name string) {
	kubeConfig := ReadKubeConfig()
	kubeConfig.CurrentContext = name
	WriteToKubeConfig(kubeConfig)
}

func SetClusterInKubeConfig(name string, serverURL string, caData []byte) {
	kubeConfig := ReadKubeConfig()
	kubeConfig.Clusters[name] = &clientcmdapi.Cluster{
		Server:                   serverURL,
		CertificateAuthorityData: caData,
	}
	WriteToKubeConfig(kubeConfig)
}

func SetUserInKubeConfig(name string, token string) {
	kubeConfig := ReadKubeConfig()
	kubeConfig.AuthInfos[name] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	WriteToKubeConfig(kubeConfig)
}

func SetContextInKubeConfig(name string, clusterName string, userName string) {
	kubeConfig := ReadKubeConfig()
	kubeConfig.Contexts[name] = &clientcmdapi.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	WriteToKubeConfig(kubeConfig)
}

func ReadKubeConfig() *clientcmdapi.Config {
	// Obtain the default kubeconfig's location
	kubeConfigLoc := GetKubeConfigLocation()

	// Load the default kubeconfig's configuration into clientcmdapi object
	kubeConfig, err := clientcmd.LoadFromFile(kubeConfigLoc)
	if err != nil {
		panic("Unable to read kubeconfig")
	}
	return kubeConfig
}

func WriteToKubeConfig(kubeConfig *clientcmdapi.Config) {
	// Write the new configuration into the default kubeconfig file
	kubeConfigLoc := GetKubeConfigLocation()

	err := clientcmd.WriteToFile(*kubeConfig,
		kubeConfigLoc,
	)
	if err != nil {
		panic("Unable to write to kubeconfig")
	}
}

func GetCurrentContextFromKubeConfig() string {
	kubeConfig := ReadKubeConfig()
	return kubeConfig.CurrentContext
}

