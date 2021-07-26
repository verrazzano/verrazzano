// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	pfake "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

// Unit test for checking output
func TestNewCmdProjectListRolesOutput(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient:  fake.NewSimpleClientset(),
		fakeClustersClient: pfake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectListRoles(streams, fakeKubernetes)

	asserts.NoError(testCmd.Execute())
}
