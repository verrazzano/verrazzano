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

var (
	argumentsData = []struct {
		args []string
	}{{[]string{"a"}},
		{[]string{"a", "b"}},
		{[]string{"a", "b", "c"}},
	}
)

// TestNewCmdProjectList_Arguments ensures no arguments are passed to project list command
func TestNewCmdProjectList_Arguments(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectList(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	// Creating projects to display them
	for _, argument := range argumentsData {
		addCmd.SetArgs(argument.args)
		err := addCmd.Execute()
		if err != nil {
			return
		}
	}

	// Calling with no arguments should not throw an error
	testCmd.SetArgs(nil)
	asserts.NoError(testCmd.Execute())

	// Calling with arguments should throw an error
	for _, argument := range argumentsData {
		testCmd.SetArgs(argument.args)
		err := testCmd.Execute()
		asserts.EqualError(err, fmt.Sprintf("accepts 0 arg(s), received %d", len(argument.args)))
	}
}

func TestNewCmdProjectList_Output(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectList(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)
	for _, argument := range singleProjects {
		addCmd.SetArgs([]string{argument}) // Need to create projects before we successfully attempt to list them
		err2 := addCmd.Execute()
		if err2 != nil {
			return
		}
		outBuffer.Reset()
	}

	expected := "NAME       AGE    CLUSTERS   NAMESPACES   \nproject1   292y   local      project1     \nproject2   292y   local      project1     \nproject3   292y   local      project1     \n\n"
	testCmd.SetArgs(nil)
	err := testCmd.Execute()
	asserts.NoError(err)
	asserts.Equal(expected, outBuffer.String())
	outBuffer.Reset()
}
