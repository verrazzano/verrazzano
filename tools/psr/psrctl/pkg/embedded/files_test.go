// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package embedded

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test(t *testing.T) {
	tmpDir, err := CreatePsrTempDir()
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	assertDirExists(t, tmpDir)

	man, err := NewPsrManifests(tmpDir)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	assertDirExists(t, man.ScenarioAbsDir)
	assertDirExists(t, man.UseCasesAbsDir)
	assertDirExists(t, man.WorkerChartAbsDir)
}

func assertDirExists(t *testing.T, dir string) {

}
