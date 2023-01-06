// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package manifest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEmbeddedManifests tests the createPsrTempDir and newPsrManifests functions
// GIVEN a binary with the embedded manifests
//
//	WHEN the PsrManifest is created
//	THEN ensure that the resulting directories exist
func TestEmbeddedManifests(t *testing.T) {
	tmpDir, err := createPsrTempDir()
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	assertDirExists(t, tmpDir)

	man, err := newPsrManifests(tmpDir)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	assertDirExists(t, man.ScenarioAbsDir)
	assertDirExists(t, man.UseCasesAbsDir)
	assertDirExists(t, man.WorkerChartAbsDir)
}

func assertDirExists(t *testing.T, dir string) {
	_, err := os.Stat(dir)
	assert.NoError(t, err)
}
