// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	testclient "k8s.io/client-go/rest/fake"
	"testing"
)

// TestExecPod tests running a command on a remote pod
// GIVEN a pod in a cluster and a command to run on that pod
//  WHEN ExecPod is called
//  THEN ExecPod return the stdout, stderr, and a nil error
func TestExecPod(t *testing.T) {
	NewSPDYExecutor = FakeNewSPDYExecutor
	FakeStdOut = "foobar"
	cfg, _ := rest.InClusterConfig()
	client := &testclient.RESTClient{}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}
	stdout, _, err := ExecPod(cfg, client, pod, "container", []string{"run", "some", "command"})
	assert.Nil(t, err)
	assert.Equal(t, FakeStdOut, stdout)
}
