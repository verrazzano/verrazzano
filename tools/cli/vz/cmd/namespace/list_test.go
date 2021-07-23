// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

// unit test for checking output
func TestNewCmdNamespaceListOutput(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, errBuffer := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceList(streams, fakeKubernetes)
	createCmd := NewCmdNamespaceCreate(streams, fakeKubernetes)

	// Creating ns
	for _, n := range singleNs {
		createCmd.SetArgs([]string{n})
		asserts.NoError(createCmd.Execute())
	}
	outBuffer.Reset()
	errBuffer.Reset()

	// verify output
	asserts.NoError(testCmd.Execute())
	asserts.Equal(outBuffer.String(),"")
}
