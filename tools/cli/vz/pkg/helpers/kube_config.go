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

// Helper function to obtain the default kubeconfig location
func GetKubeconfigLocation() (string,error) {

	var kubeconfig string
	kubeconfigEnvVar := os.Getenv("KUBECONFIG")

	if len(kubeconfigEnvVar) > 0 {
		// Find using environment variables
		kubeconfig = kubeconfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		return "", errors.New("Could not find kube config")
	}
	return kubeconfig,nil
}

func RemoveCluster(kubeconfig *clientcmdapi.Config, name string) {
	delete(kubeconfig.Clusters, name)
}

func RemoveUser(kubeconfig *clientcmdapi.Config, name string) {
	delete(kubeconfig.AuthInfos,"verrazzano")
}

func RemoveContext(kubeconfig *clientcmdapi.Config, name string) {
	delete(kubeconfig.Contexts,name)
}

func SetCurrentContext(kubeconfig *clientcmdapi.Config, name string){
	kubeconfig.CurrentContext = name
}

func SetCluster(kubeconfig *clientcmdapi.Config,name string, server_url string, caData []byte) {
	kubeconfig.Clusters[name] = &clientcmdapi.Cluster{
	Server: server_url,
	CertificateAuthorityData: caData,
	}
}

func SetUser(kubeconfig *clientcmdapi.Config, name string, token string) {
	kubeconfig.AuthInfos[name] = &clientcmdapi.AuthInfo{
		Token: token,
	}
}

func SetContext(kubeconfig *clientcmdapi.Config, name string, cluster_name string, user_name string) {
	kubeconfig.Contexts[name] = &clientcmdapi.Context{
		Cluster: cluster_name,
		AuthInfo: user_name,
	}
}

