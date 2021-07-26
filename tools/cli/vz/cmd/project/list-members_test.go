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

func TestNewCmdProjectListMembers(t *testing.T) {
	asserts := assert.New(t)
	fakeKubernetes := &TestKubernetes{
		fakeProjectClient: fake.NewSimpleClientset(),
		fakek8sClient:     k8sfake.NewSimpleClientset(),
	}

	// NewTestIOStreams returns a valid IOStreams and in, out, errout buffers for unit tests
	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdProjectListMembers(streams, fakeKubernetes)
	addCmd := NewCmdProjectAdd(streams, fakeKubernetes)
	addMemberCmd := NewCmdProjectAddMemberRole(streams, fakeKubernetes)

	// creating a project & displaying (non-existent) member roles
	for i := range singleProjects {
		addCmd.SetArgs([]string{singleProjects[i]})
		addCmd.Execute()
		outBuffer.Reset()
		projectFlag := flag.String("project-name", singleProjects[i], "project to display member-roles of")
		testCmd.Flag(*projectFlag)
		testCmd.ParseFlags([]string{*projectFlag})
		testCmd.SetArgs(argAddMember[i])
		asserts.NoError(testCmd.Execute())
		asserts.Equal(outBuffer.String(), "no members present")
		outBuffer.Reset()
	}

	// adding member roles to previously defined projects
	for i := range argAddMember {
		projectFlag := flag.String("project-name", singleProjects[i], "project to add-member-role")
		addMemberCmd.ParseFlags([]string{*projectFlag})
		addMemberCmd.Flag(*projectFlag)
		addMemberCmd.SetArgs(argAddMember[i])
		addMemberCmd.Execute()
	}
	outBuffer.Reset()

	for i := range singleProjects {
		projectFlag := flag.String("project-name", singleProjects[i], "project to display member-roles of")
		testCmd.ParseFlags([]string{*projectFlag})
		testCmd.Flag(*projectFlag)
		asserts.NoError(testCmd.Execute())
		outBuffer.Reset()
	}
}
