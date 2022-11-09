// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestGetHTTPClient(t *testing.T) {
	httpClient := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetHTTPClient()
	assert.NotNil(t, httpClient)
}

func TestGetOutputStream(t *testing.T) {
	outputStream := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetOutputStream()
	assert.NotNil(t, outputStream)
}

func TestGetInputStream(t *testing.T) {
	inputStream := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetInputStream()
	assert.NotNil(t, inputStream)
}

func TestGetErrorStream(t *testing.T) {
	inputStream := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetErrorStream()
	assert.NotNil(t, inputStream)
}

func TestGetKubeConfigGivenCommand(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, "/tmp/kubeconfig", "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, "testcontext", "")
	_, err := getKubeConfigGivenCommand(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

func TestGetClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, "/tmp/kubeconfig", "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, "testcontext", "")
	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

func TestGetKubeClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, "/tmp/kubeconfig", "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, "testcontext", "")
	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetKubeClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

func TestGetDynamicClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, "/tmp/kubeconfig", "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, "testcontext", "")
	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetDynamicClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}
