// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/cli/vz/cmd/project"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var projectNs = []string{"project1","project2","project3"}
var moveData = [][]string{{"ns1","project2"},{"ns2","project3"},{"ns3","project1"}}

// Unit test for moving a verrazzano namespace to a project
func TestNewCmdNamespaceMoveDefault(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _:= genericclioptions.NewTestIOStreams()
	testCmd := NewCmdNamespaceMove(streams, fakeKubernetes)
	createCmd := NewCmdNamespaceCreate(streams,fakeKubernetes)
	projectaddCmd := project.NewCmdProjectAdd(streams,fakeKubernetes)

	for _, ns := range singleNs {
		createCmd.SetArgs([]string{ns})
		createCmd.Execute()
	}
	for _, args := range projectNs {
		projectaddCmd.SetArgs([]string{args})
		asserts.NoError(projectaddCmd.Execute())
		projectaddCmd.Execute()
	}
	outBuffer.Reset()

	for _, args := range moveData {
		testCmd.SetArgs(args)
		asserts.NoError(testCmd.Execute())
	}
	actual :=`"ns1" namespace moved to "project2" project
"ns2" namespace moved to "project3" project
"ns3" namespace moved to "project1" project
`
	asserts.Equal(outBuffer.String(),actual)
}
