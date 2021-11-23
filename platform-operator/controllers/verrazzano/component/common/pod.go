// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"bytes"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
)

// NewSPDYExecutor is to be overridden during unit tests
var NewSPDYExecutor = remotecommand.NewSPDYExecutor

// FakeStdOut can be used to output arbitrary strings during unit testing
var FakeStdOut = ""

//FakeNewSPDYExecutor should be used instead of remotecommand.NewSPDYExecutor in unit tests
func FakeNewSPDYExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return &fakeExecutor{method: method, url: url}, nil
}

// fakeExecutor is for unit testing
type fakeExecutor struct {
	method string
	url    *url.URL
}

// Stream on a fakeExecutor sets stdout to FakeStdOut
func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString(FakeStdOut)
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

//ExecPod runs a remote command a pod, returning the stdout and stderr of the command.
func ExecPod(cfg *rest.Config, restClient rest.Interface, pod *v1.Pod, command []string) (string, string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	request := restClient.
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	executor, err := NewSPDYExecutor(cfg, "POST", request.URL())
	if err != nil {
		return "", "", err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return "", "", errors.New(fmt.Sprintf("error running command %s on %v/%v: %v", command, pod.Namespace, pod.Name, err))
	}

	return stdout.String(), stderr.String(), nil
}
