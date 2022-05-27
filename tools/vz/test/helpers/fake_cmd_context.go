// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeRootCmdContext struct {
	client client.Client
	genericclioptions.IOStreams
}

// GetOutputStream - return the output stream
func (rc *FakeRootCmdContext) GetOutputStream() io.Writer {
	return rc.IOStreams.Out
}

// GetErrorStream - return the error stream
func (rc *FakeRootCmdContext) GetErrorStream() io.Writer {
	return rc.IOStreams.ErrOut
}

// GetInputStream - return the input stream
func (rc *FakeRootCmdContext) GetInputStream() io.Reader {
	return rc.IOStreams.In
}

// GetClient - return a kubernetes client that supports the schemes used by the CLI
func (rc *FakeRootCmdContext) GetClient(cmd *cobra.Command) (client.Client, error) {
	return rc.client, nil
}

// SetClient - set the client
func (rc *FakeRootCmdContext) SetClient(client client.Client) {
	rc.client = client
}

func NewFakeRootCmdContext(streams genericclioptions.IOStreams) *FakeRootCmdContext {
	return &FakeRootCmdContext{
		IOStreams: streams,
	}
}
