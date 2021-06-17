// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

//1) Different permutation of arguments
//2) Output formatting
//3) Main Logic Testing - Might want to use fake types

var (
	description = "-d test managed cluster"
	prometheus = "-p test-prometheus-secret"
	out = "verrazzanomanagedcluster/test%d created\n"
	tests = []struct {
		args []string
		output string
	} {
		{
			[]string{"test1", description, prometheus},
			fmt.Sprintf(out, 1),
		},
		{
			[]string{"test2", prometheus, description},
			fmt.Sprintf(out, 2),
		},
		{
			[]string{prometheus, description, "test3"},
			fmt.Sprintf(out, 3),
		},
		{
			[]string{prometheus, "test4", description},
			fmt.Sprintf(out, 4),
		},
		{
			[]string{description, prometheus, "test5"},
			fmt.Sprintf(out, 5),
		},
		{
			[]string{description, "test6", prometheus},
			fmt.Sprintf(out, 6),
		},
	}
)

type TestKubernetes struct {}

func (o *TestKubernetes) NewClientSet() (clientset.Interface, error) {
	return fake.NewSimpleClientset(), nil
}

func (o *TestKubernetes) NewKubernetesClientSet() kubernetes.Interface {
	return k8sfake.NewSimpleClientset()
}

//Test different permutations for flags for register command
func TestNewCmdClusterRegister (t *testing.T) {

	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{}

	//NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdClusterRegister(streams, fakeKubernetes)

	for _, test := range tests {
		testCmd.SetArgs(test.args)
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(test.output, outBuffer.String())
		outBuffer.Reset()
	}
}

func TestNewCmdClusterList (t *testing.T) {

	assrt := assert.New(t)
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testKubernetes := &TestKubernetes{}
	testCmd := NewCmdClusterList(streams, testKubernetes)

	//create fake vmcs
	testRegisterCmd := NewCmdClusterRegister(streams, testKubernetes)

	for _, test := range tests {
		testRegisterCmd.SetArgs(test.args)
		err := testRegisterCmd.Execute()
		assrt.NoError(err)
	}
	outBuffer.Reset()

	err := testCmd.Execute()
	assrt.NoError(err)
	//fmt.Println(outBuffer.String())
}

//func buildKubeConfig(kubeconfig string) *restclient.Config {
//	var err error
//	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
//	if err != nil {
//		ginkgo.Fail("Could not get current context from kubeconfig " + kubeconfig)
//	}
//
//	return config
//}
