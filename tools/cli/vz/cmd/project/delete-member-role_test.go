// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

// Commenting out below code as add-member-role test.go is not implemented completely.

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/fake"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var argDeleteMember = []string{"verrazzano-project-monitor", "verrazzano-project-monitor", "verrazzano-project-admin"}

func TestNewCmdProjectDeleteMemberRole(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectDeleteMemberRole(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)
	addMemberCmd := NewCmdProjectAddMemberRole(streams, fakeKubernetes)

	// creating a project
	for _, project := range singleProjects {
		addCmd.SetArgs([]string{project})
		addCmd.Execute()
	}
	// adding member roles for those respective projects
	for i, strings := range argAddMember {
		addMemberCmd.SetArgs([]string{
			fmt.Sprintf("--project-name=%s", singleProjects[i]),
			strings[0],
			strings[1],
		})
		addMemberCmd.Execute()
	}
	outBuffer.Reset()

	// deleting member roles for those projects
	for i := range argDeleteMember {
		testCmd.SetArgs([]string{
			fmt.Sprintf("--project-name=%s", singleProjects[i]),
			argDeleteMember[i],
		})
		asserts.NoError(testCmd.Execute())
		outBuffer.Reset()
	}
}
