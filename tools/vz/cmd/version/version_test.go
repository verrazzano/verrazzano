// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// TestVersionCmd - check that command reports not implemented yet
func TestVersionCmd(t *testing.T) {

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	versionCmd := NewCmdVersion(rc)
	assert.NotNil(t, versionCmd)

	// Run version command, check for the expected status results to be displayed
	err := versionCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	assert.Equal(t, "Not implemented yet\n", result)
}
