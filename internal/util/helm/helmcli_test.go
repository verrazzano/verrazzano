// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

const ns = "my_namespace"
const chartdir = "my_charts"
const release = "my_release"

// goodRunner is used to test helm without actually running an OS exec command
type goodRunner struct{
	t *testing.T
}

// TestUpgrade tests the Helm upgrade command
// GIVEN a set of upgrade parameters
//  WHEN I call Upgrade
//  THEN the Helm upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(goodRunner{t: t})

	stdout, stderr, err := Upgrade(release, ns, chartdir)
	assert.NoError(err,"Upgrade returned an error")
	assert.Len(stderr,0,"Upgrade stderr should be empty")
	assert.NotZero(stdout,"Upgrade stdout should not be empty")

}

// RunCommand should assert that the cmd contains the correct data
func (r goodRunner) Run (cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains( cmd.Path, "helm","exec command should contain helm")
	assert.Contains( cmd.Args[0], "helm","exec args should contain helm")
	assert.Contains( cmd.Args[1], "upgrade","exec args should contain upgrade")
	assert.Contains( cmd.Args[2], release,"exec args should contain release name")
	assert.Contains( cmd.Args[3], chartdir,"exec args should contain chart dir ")

	return []byte("success"), []byte(""), nil
}



