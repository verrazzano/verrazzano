// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var argAddMember = [][]string{
	{"user1", "verrazzano-project-monitor"},
	{"user2", "verrazzano-project-monitor"},
	{"user3", "verrazzano-project-admin"},
}

func TestNewCmdProjectAddMemberRole(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()

	testCmd := NewCmdProjectAddMemberRole(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)

	// creating projects for member roles to
	for _, project := range singleProjects {
		addCmd.SetArgs([]string{project})
		asserts.NoError(addCmd.Execute())
	}
	outBuffer.Reset()

	// fetching and adding member roles to the projects
	for i, arg := range argAddMember {
		testCmd.SetArgs([]string{
			fmt.Sprintf("--project-name=%s", singleProjects[i]),
			arg[0],
			arg[1],
		})
		asserts.NoError(testCmd.Execute())
		actual := `member role "` + arg[1] + `"` + " added\n"
		asserts.Equal(outBuffer.String(), actual)
		outBuffer.Reset()
	}
}
