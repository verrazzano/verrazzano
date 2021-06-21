// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

//1) Different permutation of arguments
//2) Output formatting
//3) Main Logic Testing - Might want to use fake types

var (
	description    = "-d test managed cluster"
	caSecret       = "-c test-caSecret-secret"
	registerOutput = "verrazzanomanagedcluster/test%d created\n"
	tests          = []struct {
		args []string
		output string
	} {
		{
			[]string{"test1", description, caSecret},
			fmt.Sprintf(registerOutput, 1),
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
	fakeClient clientset.Interface
	fakek8sClient kubernetes.Interface
}

func (o *TestKubernetes) NewClientSet() (clientset.Interface, error) {
	return o.fakeClient, nil
}

func (o *TestKubernetes) NewKubernetesClientSet() kubernetes.Interface {
	return o.fakek8sClient
}

//Test different permutations for flags for register command
func TestNewCmdClusterRegister (t *testing.T) {

	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeClient: fake.NewSimpleClientset(),
	}

	//NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdClusterRegister(streams, fakeKubernetes)

	//creation of vmcs with proper arguments should not raise an error
	for _, test := range tests {
		testCmd.SetArgs(test.args)
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(test.output, outBuffer.String())
		outBuffer.Reset()
	}

	//creation of vmc with name that already exists should raise an error
	testCmd.SetArgs(tests[0].args)
	err := testCmd.Execute()
	asserts.EqualError(err, "verrazzanomanagedclusters.clusters \"test1\" already exists")

	//creation of vmc without specifying caSecret should raise an error
	//create new cmd to reset flags
	testCmd = NewCmdClusterRegister(streams, fakeKubernetes)
	testCmd.SetArgs([]string{"test7"})
	err = testCmd.Execute()
	asserts.EqualError(err, "CA secret is needed")
}

//Current test fails because fake types are using wrong group
func TestNewCmdClusterList (t *testing.T) {

	asserts := assert.New(t)
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testKubernetes := &TestKubernetes{
		fakeClient: fake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterList(streams, testKubernetes)

	//Executing list while no vmcs exists should not give an error
	//Instead no resource found message
	err := testCmd.Execute()
	asserts.NoError(err)
	asserts.Equal(helpers.NothingFound + "\n", outBuffer.String())

	//create fake vmcs
	testRegisterCmd := NewCmdClusterRegister(streams, testKubernetes)

	for _, test := range tests[:3] {
		testRegisterCmd.SetArgs(test.args)
		err := testRegisterCmd.Execute()
		asserts.NoError(err)
	}
	outBuffer.Reset()

	//There are 3 spaces after the each column(last one as well)
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
		fakeClient: fake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterDeregister(streams, testKubernetes)

	//trying to deregister a cluster which doesn't exists should raise an error
	testCmd.SetArgs([]string{"test"})
	err := testCmd.Execute()
	asserts.EqualError(err, "verrazzanomanagedclusters.clusters \"test\" not found")

	//register a vmc and then deregister it
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
		fakeClient: fake.NewSimpleClientset(),
		fakek8sClient: k8sfake.NewSimpleClientset(),
	}
	testCmd := NewCmdClusterManifest(streams, testKubernetes)

	//trying to get manifest secret for non existing cluster raises an error
	testCmd.SetArgs([]string{"test"})
	err := testCmd.Execute()
	asserts.EqualError(err, "verrazzanomanagedclusters.clusters \"test\" not found")

	//create a vmc resource
	testCmdRegister := NewCmdClusterRegister(streams, testKubernetes)
	testCmdRegister.SetArgs(tests[0].args)
	err = testCmdRegister.Execute()
	asserts.NoError(err)
	outBuffer.Reset()

	//trying to fetch the manifest for a vmc that exists can
	//raise an error if the manifest is not yet generated
	testCmd.SetArgs([]string{tests[0].args[0]})
	err = testCmd.Execute()
	asserts.EqualError(err, "secrets \"\" not found")

	//fake clients don't generate a manifest
	//so create a fake manifest for test
	asserts.NoError(newFakeSecret(testKubernetes.fakek8sClient))
	testCmd.SetArgs([]string{tests[0].args[0]})
	err = testCmd.Execute()
	asserts.NoError(err)

	outBuffer.Reset()
}

func newFakeSecret (fakek8sClient kubernetes.Interface) error {
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
		Data: map[string][]byte {
			"yaml" : []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL"),
		},
	}
	_, err := fakek8sClient.CoreV1().Secrets(vmcNamespace).Create(context.Background(), fakeSecret, metav1.CreateOptions{})
	return err
}