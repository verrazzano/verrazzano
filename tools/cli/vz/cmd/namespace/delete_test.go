// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"strconv"
	"testing"
)

// unit test for checking arguments
// unit test for deleting non-existent namespaces

func TestNewCmdNamespaceDeleteArgument(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceDelete(streams, fakeKubernetes)
	createCmd := NewCmdNamespaceCreate(streams, fakeKubernetes)

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

	// Calling with 1 namespace should not throw an error
	for _, n := range singleNs {
		createCmd.SetArgs([]string{n})
		createCmd.Execute()
		outBuffer.Reset()
		testCmd.SetArgs([]string{n})
		asserts.NoError(testCmd.Execute())
		asserts.Equal(outBuffer.String(), `namespace "`+n+`"`+" deleted\n")
		outBuffer.Reset()
	}
}

func TestNewCmdNamespaceDeleteDNE(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, _, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceDelete(streams, fakeKubernetes)

	// Calling with 1 namespace should not throw an error
	for _, n := range singleNs {
		testCmd.SetArgs([]string{n})
		asserts.Error(testCmd.Execute())
		asserts.Equal(errBuffer.String(), `namespaces "`+n+`"`+" not found\n")
		errBuffer.Reset()
	}
}
