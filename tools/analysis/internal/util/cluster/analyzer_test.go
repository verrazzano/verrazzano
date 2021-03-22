// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"go.uber.org/zap"
	"strings"
	"testing"
)

// TestRunAnalysisBad Tests the main RunAnalysis function
// GIVEN a call to RunAnalysis
// WHEN with invalid inputs
// THEN errors are generated as expected
func TestRunAnalysisBadArgs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// Call runAnalysis with a directory that doesn't exist
	err := RunAnalysis(logger, "")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "runAnalysis failed examining directories"))

	// Call runAnalysis with a directory that exists but has no cluster roots underneath it
	err = RunAnalysis(logger, "../../../test/cluster/image-pull-case1/bobs-books")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "runAnalysis didn't find any clusters to analyze"))

	// Call runAnalysis with an analyzer that fails, it will NOT return an error here, we
	// log them as errors and continue on
	clusterAnalysisFunctions["bad-tester"] = badTestAnalyzer
	err = RunAnalysis(logger, "../../../test/cluster/image-pull-case1")
	delete(clusterAnalysisFunctions, "bad-tester")
	assert.Nil(t, err)
}

func badTestAnalyzer(log *zap.SugaredLogger, clusterRoot string) (err error) {
	return errors.New("test failure")
}
