// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"bytes"
	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	mocks "github.com/verrazzano/verrazzano/tools/cli/vz/mock"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"testing"
)

//1) Different permutation of arguments
//2) Output formatting
//3) Main Logic Testing - Might want to use fake types

//Test different permutations for flags for register command
func TestRegisterFlags (t *testing.T) {

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mockKubernetes := mocks.NewMockKubernetes(mocker)

	asserts.NotNil(mockKubernetes)

	c := fake.NewSimpleClientset()

	mockKubernetes.EXPECT().
		NewClientSet().
		Return(c, nil).AnyTimes()

	//NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdClusterRegister(streams, mockKubernetes)
	testCmd.SetOut(outBuffer)

	description := "test managed cluster"
	prometheus := "test-prometheus-secret"
	expected := `verrazzanomanagedcluster/test1 created
verrazzanomanagedcluster/test2 created
verrazzanomanagedcluster/test3 created
verrazzanomanagedcluster/test4 created
`

	//check different permutation for arguments
	testCmd.SetArgs([]string{"test1", "-d", description, "-p", prometheus})
	err := testCmd.Execute()
	asserts.NoError(err)

	testCmd.SetArgs([]string{"test2", "-p", prometheus, "-d", description})
	err = testCmd.Execute()
	assert.NoError(t, err)

	testCmd.SetArgs([]string{"-p", prometheus, "test3", "-d", description})
	err = testCmd.Execute()
	assert.NoError(t, err)

	testCmd.SetArgs([]string{"-d", description, "-p", prometheus, "test4"})
	err = testCmd.Execute()
	assert.NoError(t, err)

	mocker.Finish()
	asserts.Equal(expected, outBuffer.String())
}

func TestListOutput (t *testing.T) {
	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	cmd := NewCmdClusterList(streams)

	bufferWrite := bytes.NewBufferString("")

	cmd.SetOut(bufferWrite)
	err := cmd.Execute()
	assert.NoError(t, err)
	resultasbyte, _ := ioutil.ReadAll(bufferWrite)
	result := string(resultasbyte)

	expected :=`NAME       AGE    DESCRIPTION   APISERVER                                        
managed1   2d3h                 https://verrazzano.default.172.19.0.211.nip.io   

`
	assert.Equal(t, expected, result)
}

func buildKubeConfig(kubeconfig string) *restclient.Config {
	var err error
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		ginkgo.Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	return config
}

func newkubeconfig() string {
	var fakeconfig string
	fakeconfig = `apiVersion: v1
clusters:
- cluster:
	certificate-authority-data: LFhdfweoJFHSJKS
	server: https://1.2.3.4/1234
	name: kind-test-admin
contexts:
- context:
	cluster: kind-test-admin
	user: kind-test-admin
	name: kind-test-admin
current-context: kind-test-admin
kind: Config
preferences: {}
users:
- name: kind-test-admin
	user:
		client-certificate-data: LFhdfweoJFHSJKS
		client-key-data: LFhfefhajf
`
	return fakeconfig
}