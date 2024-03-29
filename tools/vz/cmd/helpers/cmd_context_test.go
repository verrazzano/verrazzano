// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	testKubeConfig = "/tmp/kubeconfig"
	testK8sContext = "testcontext"
)

// TestGetHTTPClient tests the functionality to return the right HTTP client.
func TestGetHTTPClient(t *testing.T) {
	rc := helpers.NewFakeRootCmdContextWithFiles(t)
	defer helpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	httpClient := rc.GetHTTPClient()
	assert.NotNil(t, httpClient)
}

// TestGetOutputStream tests the functionality to return the output stream set in the command context.
func TestGetOutputStream(t *testing.T) {
	rc := helpers.NewFakeRootCmdContextWithFiles(t)
	defer helpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	outputStream := rc.GetOutputStream()
	assert.NotNil(t, outputStream)
}

// TestGetInputStream tests the functionality to return the input stream set in the command context.
func TestGetInputStream(t *testing.T) {
	rc := helpers.NewFakeRootCmdContextWithFiles(t)
	defer helpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	inputStream := rc.GetInputStream()
	assert.NotNil(t, inputStream)
}

// TestGetInputStream tests the functionality to return the input stream set in the command context.
func TestGetErrorStream(t *testing.T) {
	rc := helpers.NewFakeRootCmdContextWithFiles(t)
	defer helpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	errorStream := rc.GetErrorStream()
	assert.NotNil(t, errorStream)
}

// TestGetKubeConfigGivenCommand tests the functionality to return the kube config set in the command context.
func TestGetKubeConfigGivenCommand(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, testKubeConfig, "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	_, err := getKubeConfigGivenCommand(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

// TestGetClient tests the functionality to return the go client based on the kubeconfig parameters set in the command context.
func TestGetClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, testKubeConfig, "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, testK8sContext, "")

	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

// TestGetClient tests the functionality to return the kube client based on the kubeconfig parameters set in the command context.
func TestGetKubeClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, testKubeConfig, "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetKubeClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}

// TestGetClient tests the functionality to return the dynamic client based on the kubeconfig parameters set in the command context.
func TestGetDynamicClient(t *testing.T) {
	cmdWithKubeConfigAndCtx := getCommandWithoutFlags()
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagKubeConfig, testKubeConfig, "")
	cmdWithKubeConfigAndCtx.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	_, err := NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).GetDynamicClient(cmdWithKubeConfigAndCtx)
	assert.Error(t, err)
}
