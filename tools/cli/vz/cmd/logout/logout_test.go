// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logout

import (
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
	originalKubeConfigLocation, err := helpers.GetKubeConfigLocation()
	asserts.NoError(err)
	originalKubeConfig, err := os.Open(originalKubeConfigLocation)
	asserts.NoError(err)
	fakeKubeConfig, err := os.Create("fakekubeconfig")
	asserts.NoError(err)
	defer os.Remove("fakekubeconfig")
	io.Copy(fakeKubeConfig, originalKubeConfig)
	originalKubeConfig.Close()
	fakeKubeConfig.Close()
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	kubeConfigBeforeLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	// Add fake clusters,usernames,contexts..
	verrazzanoAPIURL := "verrazzano.fake.nip.io/12345"
	fakeCAData := []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL")
	fakeAccessToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"
	kubeconfig, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	helpers.SetCluster(kubeconfig,
		"verrazzano",
		verrazzanoAPIURL,
		fakeCAData,
	)

	helpers.SetUser(kubeconfig,
		"verrazzano",
		fakeAccessToken,
	)

	helpers.SetContext(kubeconfig,
		"verrazzano"+"@"+kubeconfig.CurrentContext,
		"verrazzano",
		"verrazzano",
	)

	helpers.SetCurrentContext(kubeconfig,
		"verrazzano"+"@"+kubeconfig.CurrentContext,
	)
	err = clientcmd.WriteToFile(*kubeconfig,
		"fakekubeconfig",
	)
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")

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

}

// To test logout with out logging-in
func TestRepeatedLogout(t *testing.T) {

	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	originalKubeConfigLocation, err := helpers.GetKubeConfigLocation()
	asserts.NoError(err)
	originalKubeConfig, err := os.Open(originalKubeConfigLocation)
	asserts.NoError(err)
	fakeKubeConfig, err := os.Create("fakekubeconfig")
	asserts.NoError(err)
	defer os.Remove("fakekubeconfig")
	io.Copy(fakeKubeConfig, originalKubeConfig)
	originalKubeConfig.Close()
	fakeKubeConfig.Close()
	currentDirectory, err := os.Getwd()
	asserts.NoError(err)

	kubeConfigBeforeLogout, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	os.Setenv("KUBECONFIG", currentDirectory+"/fakekubeconfig")

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

}