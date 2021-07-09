// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"testing"
)

// testdata is defined in add_test.go

func TestNewCmdProjectDelete_Arguments(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	//streams, _, _, _ := genericclioptions.NewTestIOStreams()

	testCmd := NewCmdProjectDelete(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	// Calling with no arguments should raise an error
	err := testCmd.Execute()
	asserts.EqualError(err, "accepts 1 arg(s), received 0")

	// Calling with 1 argument should not throw an error
	for _, argument := range singleProjects {
		correctOuput := "project/" + argument + " deleted\n"
		addCmd.SetArgs([]string{argument}) // Need to create projects before we successfully attempt to delete them
		testCmd.SetArgs([]string{argument})
		err2 := addCmd.Execute()
		if err2 != nil {
			fmt.Fprintf(streams.ErrOut, "project could not be added/n")
		}
		outBuffer.Reset() // Clearing project add command's output
		err := testCmd.Execute()
		asserts.NoError(err)
		asserts.Equal(correctOuput, outBuffer.String())
		outBuffer.Reset()
	}

	// Calling with multiple arguments should throw an error
	for _, arg := range multipleProjects {
		testCmd.SetArgs(arg)
		err := testCmd.Execute()
		asserts.EqualError(err, fmt.Sprintf("accepts 1 arg(s), received %d", len(arg)))
	}
}

// TestNewCmdProjectDelete_DNE verifies "project delete" command can't delete projects which don't exist
func TestNewCmdProjectDelete_DNE(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, _, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectDelete(streams, fakeKubernetes)

	// Calling delete without creating project should throw an error
	for _, argument := range singleProjects {
		testCmd.SetArgs([]string{argument})
		err := testCmd.Execute()
		asserts.Error(err)
		asserts.Equal(fmt.Sprintf("verrazzanoprojects.clusters \"%s\" not found\n", argument), errBuffer.String())
		errBuffer.Reset()
	}
}
