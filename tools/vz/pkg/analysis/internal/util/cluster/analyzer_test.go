// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"errors"
	vzhelper "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"go.uber.org/zap"
)

// TestRunAnalysisBad Tests the main RunAnalysis function
// GIVEN a call to RunAnalysis
// WHEN with invalid inputs
// THEN errors are generated as expected
func TestRunAnalysisBadArgs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := vzhelper.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	// Call runAnalysis with a directory that doesn't exist
	err := RunAnalysis(rc, logger, "")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "runAnalysis failed examining directories"))

	// Call runAnalysis with a directory that exists but has no cluster roots underneath it
	err = RunAnalysis(rc, logger, "../../../test/cluster/image-pull-case1/bobs-books")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "runAnalysis didn't find any clusters to analyze"))

	// Call runAnalysis with an analyzer that fails, it will NOT return an error here, we
	// log them as errors and continue on
	clusterAnalysisFunctions["bad-tester"] = badTestAnalyzer
	err = RunAnalysis(rc, logger, "../../../test/cluster/image-pull-case1")
	delete(clusterAnalysisFunctions, "bad-tester")
	assert.Nil(t, err)
}

// TestRunAnalysisValidArgs Tests the main RunAnalysis function
// GIVEN a call to RunAnalysis
// WHEN with valid inputs
// THEN no error is returned
func TestRunAnalysisValidArgs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := vzhelper.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})

	// Call runAnalysis with an analyzer that passes, we log the info and continue
	err := RunAnalysis(rc, logger, "../../../test/cluster/cluster-snapshot")
	assert.Nil(t, err)

}
func badTestAnalyzer(log *zap.SugaredLogger, clusterRoot string) (err error) {
	return errors.New("test failure")
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles(t *testing.T) (*os.File, *os.File) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	assert.NoError(t, err)

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	assert.NoError(t, err)

	return stdoutFile, stderrFile
}
