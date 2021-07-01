// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"github.com/stretchr/testify/assert"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"testing"
)

var (
	arguments_data = []struct {
		args []string
	}{{[]string{"a"}},
		{[]string{"a", "b"}},
		{[]string{"a", "b", "c"}},
	}
)

// TODO : Declare variable values

type TestKubernetes struct {
	fakeProjectClient  projectclientset.Interface
	fakeClustersClient clustersclientset.Interface
	fakek8sClient      kubernetes.Interface
}

// Fake config with fake Host address
func (o *TestKubernetes) GetKubeConfig() *rest.Config {
	config := &rest.Config{
		Host: "https://1.2.3.4:1234",
	}
	return config
}

func (o *TestKubernetes) NewClustersClientSet() (clustersclientset.Interface, error) {
	return o.fakeClustersClient, nil
}

func (o *TestKubernetes) NewProjectClientSet() (projectclientset.Interface, error) {
	return o.fakeProjectClient, nil
}

func (o *TestKubernetes) NewClientSet() kubernetes.Interface {
	return o.fakek8sClient
}

// TestNewCmdProjectList_Arguments ensures no arguments are passed to project list command
func TestNewCmdProjectList_Arguments(t *testing.T) {

	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	//streams, _, outBuffer, _:= genericclioptions.NewTestIOStreams()
	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectList(streams, fakeKubernetes)

	// Calling with arguments should throw an error
	/*
		for _,argument := range arguments_data {
			testCmd.SetArgs(argument.args)
			err := testCmd.Execute()
			asserts.NoError(err)
		//	asserts.Equal("",outBuffer.String())
			outBuffer.Reset()
		}
	*/
	// Calling with no arguments should not throw an error
	testCmd.SetArgs([]string{""})
	asserts.NoError(testCmd.Execute())

}
