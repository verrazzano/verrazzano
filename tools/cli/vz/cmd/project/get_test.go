// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"testing"
)

var getOutput []string = []string{
	"NAME       AGE    CLUSTERS   NAMESPACES   \nproject1   292y   local      project1     \n\n",
	"NAME       AGE    CLUSTERS   NAMESPACES   \nproject2   292y   local      project1     \n\n",
	"NAME       AGE    CLUSTERS   NAMESPACES   \nproject3   292y   local      project1     \n\n",
}

// testdata is defined in add_test.go

// Unit test for verifying CmdProjectGet fails for incorrect number of arguments
func TestNewCmdProjectGet_Arguments(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectGet(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)
	DeleteCmd := NewCmdProjectDelete(streams, fakeKubernetes)

	// Calling with no arguments should raise an error
	asserts.EqualError(testCmd.Execute(), "accepts 1 arg(s), received 0")
	errBuffer.Reset()

	// Calling with 1 argument should not throw an error
	for _, argument := range singleProjects {
		addCmd.SetArgs([]string{argument}) // Need to create projects before we successfully attempt to get them
		testCmd.SetArgs([]string{argument})
		err2 := addCmd.Execute()
		if err2 != nil {
			return
		}
		err := testCmd.Execute()
		asserts.NoError(err)
		DeleteCmd.SetArgs([]string{argument}) // Teardown i.e deleting projects after every iteration
		err = DeleteCmd.Execute()
		if err != nil {
			return
		}
		outBuffer.Reset()
	}

	// Calling with multiple arguments should throw an error
	for _, arg := range multipleProjects {
		testCmd.SetArgs(arg)
		err := testCmd.Execute()
		asserts.EqualError(err, fmt.Sprintf("accepts 1 arg(s), received %d", len(arg)))
		errBuffer.Reset()
	}
}

// Unit test for verifying CmdProjectGet prints the correct ouput for a single argument
func TestNewCmdProjectGet_Output(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOstreams returns a valid IOStreams and input,out and err buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectGet(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	for i, argument := range singleProjects {
		addCmd.SetArgs([]string{argument}) // Need to create projects before we successfully attempt to delete them
		testCmd.SetArgs([]string{argument})
		err2 := addCmd.Execute()
		if err2 != nil {
			fmt.Fprintf(streams.ErrOut, "project could not be added/n")
		}
		outBuffer.Reset() // Clearing project add command's output
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(getOutput[i], outBuffer.String())
		outBuffer.Reset()
	}
}
