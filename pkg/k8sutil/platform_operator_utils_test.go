// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package k8sutil_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
)

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestGetInstalledBOMData tests executing into the Platform Operator pod and returns the installed BOM file data as JSON
// GIVEN a kubeconfig
//
//	WHEN GetInstalledBOMData is called
//	THEN GetInstalledBOMData return nil error
func TestGetInstalledBOMData(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	stdout := []byte("verrazzano-platform-operator")
	stdErr := []byte{}

	k8sutil.SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    nil,
	})
	defer k8sutil.SetDefaultRunner()
	_, err = k8sutil.GetInstalledBOMData(dummyKubeConfig)
	assert.Nil(t, err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

// TestGetInstalledBOMData tests executing into the Platform Operator pod and returns the installed BOM file data as JSON
// GIVEN a kubeconfig
//
//	WHEN GetInstalledBOMData is called
//	THEN GetInstalledBOMData returns error
func TestGetInstalledBOMDataFailure(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	stdout := []byte("verrazzano-platform-operator")
	stdErr := []byte{}

	k8sutil.SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    fmt.Errorf("Error running command"),
	})
	defer k8sutil.SetDefaultRunner()
	_, err = k8sutil.GetInstalledBOMData(dummyKubeConfig)
	assert.NotNil(t, err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}
