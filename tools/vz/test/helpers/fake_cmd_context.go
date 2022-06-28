// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeRootCmdContext struct {
	client     client.Client
	kubeClient kubernetes.Interface
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

// GetClient - return a controller runtime client that supports the schemes used by the CLI
func (rc *FakeRootCmdContext) GetClient(cmd *cobra.Command) (client.Client, error) {
	return rc.client, nil
}

// GetKubeClient - return a Kubernetes clientset for use with the fake go-client
func (rc *FakeRootCmdContext) GetKubeClient(cmd *cobra.Command) (kubernetes.Interface, error) {
	return rc.kubeClient, nil
}

// SetClient - set the client
func (rc *FakeRootCmdContext) SetClient(client client.Client) {
	rc.client = client
}

// RoundTripFunc - define the type for the Transport function
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip - define the implementation for the Transport function
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// GetHTTPClient - return an HTTP client for testing that always responds with a 200
// and a pre-defined list of releases
func (rc *FakeRootCmdContext) GetHTTPClient() *http.Client {
	// Predefined response for the list of releases
	releaseResponse := []github.ReleaseAsset{
		{
			TagName: "v1.3.0",
		},
		{
			TagName: "v1.2.0",
		},
		{
			TagName: "v1.3.1",
		},
	}
	jsonResp, _ := json.Marshal(releaseResponse)

	return &http.Client{
		Timeout: time.Second * 30,
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer(jsonResp)),
				Header:     http.Header{"Content-Type": {"application/json"}},
			}
		}),
	}
}

func NewFakeRootCmdContext(streams genericclioptions.IOStreams) *FakeRootCmdContext {
	return &FakeRootCmdContext{
		IOStreams:  streams,
		kubeClient: fake.NewSimpleClientset(),
	}
}
