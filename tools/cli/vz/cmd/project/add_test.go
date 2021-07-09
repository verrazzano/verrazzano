// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	verrazzanoclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"testing"
)

var (
	ns1            = "-n namespace1"
	ns2            = "-n namespace2,namspace3"
	ns4            = "-n namespace4"
	pmt            = "-p local"
	singleProjects = []string{
		"project1",
		"project2",
		"project3",
	}
	multipleProjects = [][]string{
		{"project1", "project2"},
		{"project3", "project4", "project5"},
		{"project5", "project6", "project3", "project4"},
		{"project7", "project8", "project5", "project6", "project3", "project4"},
	}
	multipleFlags = [][]string{
		{"project1", ns1, pmt},
		{"project2", pmt, ns2},
		{"project4", ns4},
		{"project5", pmt},
	}
)

type TestKubernetes struct {
	fakeProjectClient  projectclientset.Interface
	fakeClustersClient clustersclientset.Interface
	fakek8sClient      kubernetes.Interface
	fakev8oClient      verrazzanoclientset.Interface
}

func (o *TestKubernetes) NewVerrazzanoClientSet() (verrazzanoclientset.Interface, error) {
	return o.fakev8oClient, nil
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

// Unit test for verifying CmdProjectAdd takes only 1 argument
func TestNewCmdProjectAdd_Arguments(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectAdd(streams, fakeKubernetes)
	deleteCmd := NewCmdProjectDelete(streams, fakeKubernetes)

	// Calling with no arguments should raise an error
	// TODO : Refresh the errorBuffer - look how to remove the cobra errors, ask someone
	asserts.EqualError(testCmd.Execute(), "accepts 1 arg(s), received 0")
	errBuffer.Reset()

	// Calling with 1 argument should not throw an error
	for _, argument := range singleProjects {
		var correctOutput = "project/" + argument + " added\n"
		testCmd.SetArgs([]string{argument})
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(correctOutput, outBuffer.String())
		deleteCmd.SetArgs([]string{argument})
		err = deleteCmd.Execute()
		asserts.NoError(err)
		outBuffer.Reset()
	}

	// Calling with 1 argument & multiple flags in different permutations should not throw an error
	for _, args := range multipleFlags {
		correctOutput := "project/" + args[0] + " added\n"
		testCmd.SetArgs(args)
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(correctOutput, outBuffer.String())
		deleteCmd.SetArgs(args[:1])
		err2 := deleteCmd.Execute()
		if err2 != nil {
			return
		}
		outBuffer.Reset()
	}

	// Calling with multiple arguments should throw an error
	// TODO : Refresh the errorBuffer - look how to remove the cobra errors, ask someone
	for _, arg := range multipleProjects {
		testCmd.SetArgs(arg)
		err := testCmd.Execute()
		asserts.Error(err)
		errBuffer.Reset()
		outBuffer.Reset()
	}
}

// Unit Test for testing creation of duplicate projects
func TestNewCmdProjectAdd_Duplicates(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	// Calling with 1 argument the first time should not throw an error
	for _, argument := range singleProjects {
		correctOutput := "project/" + argument + " added\n"
		testCmd.SetArgs([]string{argument})
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(correctOutput, outBuffer.String())
		outBuffer.Reset()
	}

	// Trying to add duplicate projects should throw an error
	for _, argument := range singleProjects {
		testCmd.SetArgs([]string{argument})
		err := testCmd.Execute()
		asserts.Error(err)
		asserts.Equal(fmt.Sprintf("verrazzanoprojects.clusters \"%s\" already exists\n", argument), errBuffer.String())
		errBuffer.Reset()
	}
}
