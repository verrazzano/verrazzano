// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package buildlog

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"testing"
)

// TestBuildLogAnalyzer Tests the runAnalysis for buildlog
// GIVEN a call to runAnalysis
// WHEN with valid info
// THEN the analysis is successful
func TestBuildLogAnalyzer(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// Call RunAnalysis TODO: Update when implementation is added
	err := RunAnalysis(logger, "test directory")
	assert.Nil(t, err)
}
