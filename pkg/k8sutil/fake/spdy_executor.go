// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"bytes"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExecResult can be used to output arbitrary strings during unit testing
var PodExecResult = func(url *url.URL) (string, string, error) { return "", "", nil }

//NewPodExecutor should be used instead of remotecommand.NewSPDYExecutor in unit tests
func NewPodExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return &dummyExecutor{method: method, url: url}, nil
}

//dummyExecutor is for unit testing
type dummyExecutor struct {
	method string
	url    *url.URL
}

//Stream on a dummyExecutor sets stdout to PodExecResult
func (f *dummyExecutor) Stream(options remotecommand.StreamOptions) error {
	stdout, stderr, err := PodExecResult(f.url)
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString(stdout)
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	if options.Stderr != nil {
		buf := new(bytes.Buffer)
		buf.WriteString(stderr)
		if _, err := options.Stderr.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return err
}
