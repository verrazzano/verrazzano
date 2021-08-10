// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"k8s.io/client-go/tools/clientcmd"
)

func TestSetAndRemoveClusterFromKubeConfig(t *testing.T) {
	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	// Add fake clusters,usernames,contexts..
	fakeVerrazzanoAPIURL := "verrazzano.fake.nip.io/12345"
	fakeCAData := []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL")

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	// Add a cluster
	err = SetClusterInKubeConfig(KubeConfigKeywordVerrazzano,
		fakeVerrazzanoAPIURL,
		fakeCAData,
	)
	asserts.NoError(err)

	// Assert that the cluster is added
	kubeConfig, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok := kubeConfig.Clusters[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, true)

	// Remove the cluster
	err = RemoveClusterFromKubeConfig(KubeConfigKeywordVerrazzano)
	asserts.NoError(err)

	// Assert that cluster is removed
	kubeConfig, err = clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok = kubeConfig.Clusters[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, false)

	err = os.Remove("fakekubeconfig")
	asserts.NoError(err)
}

func TestSetAndRemoveContext(t *testing.T) {
	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	// Set context in kubeconfig
	err = SetContextInKubeConfig(KubeConfigKeywordVerrazzano,
		KubeConfigKeywordVerrazzano,
		KubeConfigKeywordVerrazzano,
	)
	asserts.NoError(err)

	// Assert that the context is added
	kubeConfig, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok := kubeConfig.Contexts[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, true)

	// Remove the context
	err = RemoveContextFromKubeConfig(KubeConfigKeywordVerrazzano)
	asserts.NoError(err)

	// Assert that the context is removed
	kubeConfig, err = clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok = kubeConfig.Contexts[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, false)

	err = os.Remove("fakekubeconfig")
	asserts.NoError(err)
}

func TestSetAndRemoveAuthInfo(t *testing.T) {
	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	fakeAccessToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"
	fakeRefreshToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"

	// Add a user to kubeconfig
	err = SetUserInKubeConfig(KubeConfigKeywordVerrazzano,
		fakeAccessToken,
		AuthDetails{
			AccessTokenExpTime:  9999999999,
			RefreshTokenExpTime: 9999999999,
			RefreshToken:        fakeRefreshToken,
		},
	)
	asserts.NoError(err)

	// Assert that the user is added
	kubeConfig, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok := kubeConfig.AuthInfos[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, true)

	// Remove the user
	err = RemoveUserFromKubeConfig(KubeConfigKeywordVerrazzano)
	asserts.NoError(err)

	// Assert that he is removed
	kubeConfig, err = clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)
	_, ok = kubeConfig.AuthInfos[KubeConfigKeywordVerrazzano]
	asserts.Equal(ok, false)

	err = os.Remove("fakekubeconfig")
	asserts.NoError(err)
}

func createFakeKubeConfig(asserts *assert.Assertions) {
	originalKubeConfigLocation, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	originalKubeConfig, err := os.Open(originalKubeConfigLocation)
	asserts.NoError(err)
	fakeKubeConfig, err := os.Create("fakekubeconfig")
	asserts.NoError(err)
	_, err = io.Copy(fakeKubeConfig, originalKubeConfig)
	asserts.NoError(err)
	err = originalKubeConfig.Close()
	asserts.NoError(err)
	err = fakeKubeConfig.Close()
	asserts.NoError(err)
}
