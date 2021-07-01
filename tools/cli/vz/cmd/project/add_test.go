// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"testing"
)

var correctOutput = "project created\n"

var (
	correctArguments = []struct {
		args   []string
		output string
	}{
		{[]string{"project1"}, correctOutput},
		{[]string{"project2"}, correctOutput},
		{[]string{"project3"}, correctOutput},
	}
	wrongArguments = [][]string{
		[]string{"project1", "project2"},
		[]string{"project3", "project4"},
		[]string{"project5", "project6"},
		[]string{"project7", "project8"},
	}
)

// struct and it's methods are declared in list_test.go

func TestNewCmdProjectAdd_Arguments(t *testing.T) {

	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	// Calling with 1 argument should not throw an error
	for _, argument := range correctArguments {
		testCmd.SetArgs(argument.args)
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(argument.output, outBuffer.String())
		outBuffer.Reset()
		// TODO : Check output withoutBuffer & assert statement
	}

	// Calling with multiple arguments should throw an error
	for _, arg := range wrongArguments {
		testCmd.SetArgs(arg)
		err := testCmd.Execute()
		asserts.Error(err)
	}

	// Calling with no arguments should raise an error
	//testCmd.SetArgs()
	asserts.NoError(testCmd.Execute())
}
