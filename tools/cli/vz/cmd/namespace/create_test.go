// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/stretchr/testify/assert"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	verrazzanoclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"strconv"

	//"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"testing"
)

var singleNs = [...]string{"ns1", "ns2", "ns3", "ns4"}
var mutipleNs = [...][]string{
	{"ns1", "ns2"},
	{"ns3", "ns4", "ns5"},
	{"ns6", "ns7", "ns8"},
}

type TestKubernetes struct {
	fakeProjectClient  projectclientset.Interface
	fakeClustersClient clustersclientset.Interface
	fakek8sClient      kubernetes.Interface
	fakev8oClient      verrazzanoclientset.Interface
}

func (o *TestKubernetes) NewVerrazzanoClientSet() (verrazzanoclientset.Interface, error) {
	return o.fakev8oClient, nil
}

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

// unit test for finding arguments passed.
// unit test for creating duplicate (verrazzano & non-verrazzano) namespaces

func TestNewCmdNamespaceCreateArguments(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceCreate(streams, fakeKubernetes)

	// Calling with no arguments should throw an error
	asserts.EqualError(testCmd.Execute(), "accepts 1 arg(s), received 0")
	outBuffer.Reset()
	errBuffer.Reset()

	// Calling with more than 1 namespace as argument should throw an error
	for _, ns := range mutipleNs {
		testCmd.SetArgs(ns)
		asserts.EqualError(testCmd.Execute(), `accepts 1 arg(s), received `+strconv.Itoa(len(ns)))
		errBuffer.Reset()
	}
}

func TestNewCmdNamespaceCreateDuplicate(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}
	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceCreate(streams, fakeKubernetes)

	// Calling with 1 namespace should not throw an error
	for _, n := range singleNs {
		testCmd.SetArgs([]string{n})
		asserts.NoError(testCmd.Execute())
		asserts.Equal(outBuffer.String(), `namespace/`+n+" created\n")
		outBuffer.Reset()
	}
	// Creating namespaces again should throw an error
	for _, n := range singleNs {
		testCmd.SetArgs([]string{n})
		asserts.Error(testCmd.Execute())
		asserts.Equal(errBuffer.String(), `namespaces "`+n+`"`+" already exists\n")
		errBuffer.Reset()
	}
}
