// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logout

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"io"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"testing"
)

var (
	test1 = []struct {
		args   []string
		output string
	}{
		{
			[]string{},
			"Logout successful!\n",
		},
	}
	test2 = []struct {
		args   []string
		output string
	}{
		{
			[]string{},
			"Already Logged out\n",
		},
	}
)

// To test basic logout
func TestNewCmdLogout(t *testing.T) {

	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	kubeConfigBeforeLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	// Add fake clusters,usernames,contexts..
	fakeVerrazzanoAPIURL := "verrazzano.fake.nip.io/12345"
	fakeCAData := []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL")
	fakeAccessToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"
	fakeRefreshToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	err = helpers.SetClusterInKubeConfig(helpers.KubeConfigKeywordVerrazzano,
		fakeVerrazzanoAPIURL,
		fakeCAData,
	)
	asserts.NoError(err)

	err = helpers.SetUserInKubeConfig(helpers.KubeConfigKeywordVerrazzano,
		fakeAccessToken,
		helpers.AuthDetails{
			AccessTokenExpTime:  9999999999,
			RefreshTokenExpTime: 9999999999,
			RefreshToken:        fakeRefreshToken,
		},
	)
	asserts.NoError(err)

	currentContext, err := helpers.GetCurrentContextFromKubeConfig()
	asserts.NoError(err)

	err = helpers.SetContextInKubeConfig(
		fmt.Sprintf("%v@%v", helpers.KubeConfigKeywordVerrazzano, currentContext),
		helpers.KubeConfigKeywordVerrazzano,
		helpers.KubeConfigKeywordVerrazzano,
	)
	asserts.NoError(err)

	err = helpers.SetCurrentContextInKubeConfig(fmt.Sprintf("%v@%v", helpers.KubeConfigKeywordVerrazzano, currentContext))
	asserts.NoError(err)

	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdLogout(streams)

	for _, test := range test1 {

		testCmd.SetArgs(test.args)
		asserts.NoError(testCmd.Execute())
		asserts.Equal(test.output, outBuffer.String())

		kubeConfigAfterLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
		asserts.NoError(err)
		asserts.Equal(kubeConfigAfterLogout, kubeConfigBeforeLogout)

		outBuffer.Reset()
	}
	err = os.Remove("fakekubeconfig")
	asserts.NoError(err)
}

// To test logout with out logging-in
func TestRepeatedLogout(t *testing.T) {

	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	kubeConfigBeforeLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	err = os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")
	asserts.NoError(err)

	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdLogout(streams)

	for _, test := range test2 {

		testCmd.SetArgs(test.args)
		asserts.NoError(testCmd.Execute())
		asserts.Equal(test.output, outBuffer.String())

		kubeConfigAfterLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
		asserts.NoError(err)
		asserts.Equal(kubeConfigAfterLogout, kubeConfigBeforeLogout)

		outBuffer.Reset()
	}
	err = os.Remove("fakekubeconfig")
	asserts.NoError(err)
}

func createFakeKubeConfig(asserts *assert.Assertions) {
	originalKubeConfigLocation, err := helpers.GetKubeConfigLocation()
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
