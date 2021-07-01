// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"testing"
)

// 1) Different permutation of arguments
// 2) Output formatting
// 3) Main Logic Testing - Might want to use fake types

var (
	description      = "-d test managed cluster"
	caSecret         = "-c test-caSecret-secret"
	registerOutput   = "verrazzanomanagedcluster/test%d created\n"
	configMapVerbose = `configmap/verrazzano-admin-cluster doesn't exist
creating configmap/verrazzano-admin-cluster
configmap/verrazzano-admin-cluster created
`
	// Creation of first vmc will create a configMap
	// named verrazzano-admin-cluster if it doesn't exist
	tests = []struct {
		args   []string
		output string
	}{
		{
			[]string{"test1", description, caSecret},
			fmt.Sprintf(configMapVerbose+registerOutput, 1),
		},
		{
			[]string{"test2", caSecret, description},
			fmt.Sprintf(registerOutput, 2),
		},
		{
			[]string{caSecret, description, "test3"},
			fmt.Sprintf(registerOutput, 3),
		},
		{
			[]string{caSecret, "test4", description},
			fmt.Sprintf(registerOutput, 4),
		},
		{
			[]string{description, caSecret, "test5"},
			fmt.Sprintf(registerOutput, 5),
		},
		{
			[]string{description, "test6", caSecret},
			fmt.Sprintf(registerOutput, 6),
		},
	}
)

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

// Test different permutations for flags for register command
func TestNewCmdClusterRegister(t *testing.T) {

	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdClusterRegister(streams, fakeKubernetes)

	// Creation of vmcs with proper arguments should not raise an error
	for _, test := range tests {
		testCmd.SetArgs(test.args)
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(test.output, outBuffer.String())
		outBuffer.Reset()
	}

	// Creation of vmc with name that already exists should raise an error
	testCmd.SetArgs(tests[0].args)
	err := testCmd.Execute()
	asserts.EqualError(err, `verrazzanomanagedclusters.clusters "test1" already exists`)

	// Creation of vmc without specifying caSecret should raise an error
	// Create new cmd to reset flags
	testCmd = NewCmdClusterRegister(streams, fakeKubernetes)
	testCmd.SetArgs([]string{"test7"})
	err = testCmd.Execute()
	asserts.EqualError(err, "CA secret is needed")
}

// Current test fails because fake types are using wrong group
func TestNewCmdClusterList(t *testing.T) {

	asserts := assert.New(t)
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterList(streams, testKubernetes)

	// Executing list while no vmcs exists should not give an error
	// Instead no resource found message
	err := testCmd.Execute()
	asserts.NoError(err)
	asserts.Equal(helpers.NothingFound+"\n", outBuffer.String())

	// create fake vmcs
	testRegisterCmd := NewCmdClusterRegister(streams, testKubernetes)

	for _, test := range tests[:3] {
		testRegisterCmd.SetArgs(test.args)
		err := testRegisterCmd.Execute()
		asserts.NoError(err)
	}
	outBuffer.Reset()

	// There are 3 spaces after the each column(last one as well)
	expected := `NAME    AGE    STATUS   DESCRIPTION             APISERVER   
test1   292y             test managed cluster               
test2   292y             test managed cluster               
test3   292y             test managed cluster               

`
	err = testCmd.Execute()
	asserts.NoError(err)
	asserts.Equal(expected, outBuffer.String())
	outBuffer.Reset()
}

func TestNewCmdClusterDeregister(t *testing.T) {
	asserts := assert.New(t)
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterDeregister(streams, testKubernetes)

	// Trying to deregister a cluster which doesn't exists should raise an error
	testCmd.SetArgs([]string{"test"})
	err := testCmd.Execute()
	asserts.EqualError(err, `verrazzanomanagedclusters.clusters "test" not found`)

	// Register a vmc and then deregister it
	testCmdRegister := NewCmdClusterRegister(streams, testKubernetes)
	testCmdRegister.SetArgs(tests[0].args)
	err = testCmdRegister.Execute()
	asserts.NoError(err)
	outBuffer.Reset()

	testCmd.SetArgs([]string{tests[0].args[0]})
	err = testCmd.Execute()
	asserts.NoError(err)
	asserts.Equal(fmt.Sprintf("%s deregistered\n", tests[0].args[0]), outBuffer.String())
}

func TestNewCmdClusterManifest(t *testing.T) {
	asserts := assert.New(t)
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterManifest(streams, testKubernetes)

	// Trying to get manifest secret for non existing cluster raises an error
	testCmd.SetArgs([]string{"test"})
	err := testCmd.Execute()
	asserts.EqualError(err, `verrazzanomanagedclusters.clusters "test" not found`)

	// Create a vmc resource
	testCmdRegister := NewCmdClusterRegister(streams, testKubernetes)
	testCmdRegister.SetArgs(tests[0].args)
	err = testCmdRegister.Execute()
	asserts.NoError(err)
	outBuffer.Reset()

	// Trying to fetch the manifest for a vmc that exists can
	// Raise an error if the manifest is not yet generated
	testCmd.SetArgs([]string{tests[0].args[0]})
	err = testCmd.Execute()
	asserts.EqualError(err, `secrets "" not found`)

	// Fake clients don't generate a manifest
	// so create a fake manifest for test
	asserts.NoError(newFakeSecret(testKubernetes.fakek8sClient))
	testCmd.SetArgs([]string{tests[0].args[0]})
	err = testCmd.Execute()
	asserts.NoError(err)

	outBuffer.Reset()
}

// Create a fake secret with garbage data
func newFakeSecret(fakek8sClient kubernetes.Interface) error {
	fakeSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "verrazzano-mc",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "clusters.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoMangedCluster",
				Name:       "test1",
			}},
		},
		// Garbage Data
		Data: map[string][]byte{
			"yaml": []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL"),
		},
	}
	_, err := fakek8sClient.CoreV1().Secrets(vmcNamespace).Create(context.Background(), fakeSecret, metav1.CreateOptions{})
	return err
}
