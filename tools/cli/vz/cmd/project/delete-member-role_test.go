// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package project

import (
	"flag"
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

		projectFlag := flag.String("project-name", singleProjects[i], "project to add-member-role")
		addMemberCmd.Flag(*projectFlag)
		addMemberCmd.SetArgs(strings)
		addMemberCmd.Execute()
	}
	outBuffer.Reset()

	// deleting member roles for those projects
	for i, s := range argDeleteMember {
		projectFlag := flag.String("project-name", singleProjects[i], "project to delete-member-role from")
		testCmd.Flag(*projectFlag)
		testCmd.SetArgs([]string{s})
		asserts.NoError(testCmd.Execute())
		outBuffer.Reset()
	}
}
