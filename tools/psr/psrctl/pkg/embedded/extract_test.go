// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package embedded

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestEmbeddedManifests tests the CreatePsrTempDir and NewPsrManifests functions
// GIVEN a binary with the embedded manifests
//
//	WHEN the PsrManifest is created
//	THEN ensure that the resulting directories exist
func TestEmbeddedManifests(t *testing.T) {
	tmpDir, err := createPsrTempDir()
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
	_, err := os.Stat(dir)
	assert.NoError(t, err)
}
