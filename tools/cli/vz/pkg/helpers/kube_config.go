// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"errors"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
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
		return "", errors.New("Could not find kube config")
	}
	return kubeConfig,nil
}

func RemoveCluster(kubeConfig *clientcmdapi.Config, name string) {
	delete(kubeConfig.Clusters, name)
}

func RemoveUser(kubeConfig *clientcmdapi.Config, name string) {
	delete(kubeConfig.AuthInfos,name)
}

func RemoveContext(kubeConfig *clientcmdapi.Config, name string) {
	delete(kubeConfig.Contexts,name)
}

func SetCurrentContext(kubeConfig *clientcmdapi.Config, name string){
	kubeConfig.CurrentContext = name
}

func SetCluster(kubeConfig *clientcmdapi.Config,name string, serverUrl string, caData []byte) {
	kubeConfig.Clusters[name] = &clientcmdapi.Cluster{
		Server: serverUrl,
		CertificateAuthorityData: caData,
	}
}

func SetUser(kubeConfig *clientcmdapi.Config, name string, token string) {
	kubeConfig.AuthInfos[name] = &clientcmdapi.AuthInfo{
		Token: token,
	}
}

func SetContext(kubeConfig *clientcmdapi.Config, name string, clusterName string, userName string) {
	kubeConfig.Contexts[name] = &clientcmdapi.Context{
		Cluster: clusterName,
		AuthInfo: userName,
	}
}

