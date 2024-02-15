// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const imagePullCase1 = "../../pkg/internal/test/cluster/image-pull-case1/"
const ingressIPNotFound = "../../pkg/internal/test/cluster/ingress-ip-not-found"

const loadBalancerErr = "Error syncing load balancer: failed to ensure load balancer: awaiting load balancer: context deadline exceeded"
const noIPFoundErr = "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer"
const istioIPErr = "Verrazzano install failed as no IP found for service istio-ingressgateway with type LoadBalancer"

// TestAnalyzeDefaultFromReadOnlyDir
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute from read only dir with a valid capture-dir and report-format set to "summary"
// THEN expect the command to do the analysis and generate report file into tmp dir
func TestAnalyzeDefaultFromReadOnlyDir(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	pwd, err := os.Getwd()
	assert.Nil(t, err)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, pwd+"/"+ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.SummaryReport)
	assert.Nil(t, os.Chdir("/"))
	defer os.Chdir(pwd)
	err = cmd.Execute()
	assert.Nil(t, err)
	if fileMatched, _ := filepath.Glob(os.TempDir() + "/" + constants.VzAnalysisReportTmpFile); len(fileMatched) == 1 {
		os.Remove(fileMatched[0])
		assert.NoFileExists(t, fileMatched[0])
	}
}

// TestAnalyzeCommandDetailedReport
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid capture-dir and report-format set to "detailed"
// THEN expect the command to provide the report containing all the details for one or more issues reported
func TestAnalyzeCommandDetailedReport(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.DetailedReport)
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), noIPFoundErr,
		loadBalancerErr)
	// Failures must be reported underreport file details-XXXXXX.out
	if fileMatched, _ := filepath.Glob(constants.VzAnalysisReportTmpFile); len(fileMatched) == 1 {
		os.Remove(fileMatched[0])
		assert.NoFileExists(t, fileMatched[0])
	}
}

// TestAnalyzeCommandSummaryReport
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid capture-dir and report-format set to "summary"
// THEN expect the command to provide the report containing only summary for one or more issues reported
func TestAnalyzeCommandSummaryReport(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.SummaryReport)
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.NotContains(t, string(buf), loadBalancerErr)
	assert.Contains(t, string(buf), noIPFoundErr)
	// Failures must be reported underreport file details-XXXXXX.out
	if fileMatched, _ := filepath.Glob(constants.VzAnalysisReportTmpFile); len(fileMatched) == 1 {
		os.Remove(fileMatched[0])
		assert.NoFileExists(t, fileMatched[0])
	}
}

// TestAnalyzeCommandInvalidReportFormat
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with an invalid value for report-format
// THEN expect the command to fail with an appropriate error message to indicate the issue
func TestAnalyzeCommandInvalidReportFormat(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, "invalid-report-format")
	err := cmd.Execute()
	assert.NotNil(t, err)
	buf, err := os.ReadFile(stderrFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), "\"invalid-report-format\" is not valid for flag report-format, only \"summary\" and \"detailed\" are valid")
}

// TestAnalyzeWithDefaultReportFormat
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute without report-format
// THEN expect the command to take the default value of summary for report-format and perform the analysis
func TestAnalyzeWithDefaultReportFormat(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
		if fileMatched, _ := filepath.Glob(constants.VzAnalysisReportTmpFile); len(fileMatched) == 1 {
			os.Remove(fileMatched[0])
			assert.NoFileExists(t, fileMatched[0])
		}
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.NotContains(t, string(buf), loadBalancerErr)
	assert.Contains(t, string(buf), noIPFoundErr)
}

// TestAnalyzeWithNonPermissiveReportFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with report-file in read only file location
// THEN expect the command to fail the analysis and do not create report file
func TestAnalyzeWithNonPermissiveReportFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.DetailedReport)
	cmd.PersistentFlags().Set(constants.ReportFileFlagName, "/TestAnalyzeCommandReportFileOutput")
	err := cmd.Execute()
	// Failures must not be reported as report file only has read permissions
	assert.NotNil(t, err)
	assert.NoFileExists(t, "/TestAnalyzeCommandReportFileOutput")
}

// TestAnalyzeCommandWithReportFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid report-file
// THEN expect the command to create the report file, containing the analysis report
func TestAnalyzeCommandWithReportFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
		os.Remove("TestAnalyzeCommandReportFileOutput")
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.DetailedReport)
	cmd.PersistentFlags().Set(constants.ReportFileFlagName, "TestAnalyzeCommandReportFileOutput")
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.FileExists(t, "TestAnalyzeCommandReportFileOutput")
}

// TestAnalyzeCommandInvalidCapturedDir
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with capture-dir not containing the cluster snapshot
// THEN expect the command to fail with an appropriate error message
func TestAnalyzeCommandInvalidCapturedDir(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, "../")
	err := cmd.Execute()
	assert.NotNil(t, err)
	buf, err := os.ReadFile(stderrFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), "Cluster Analyzer runAnalysis didn't find any clusters")
}

// TestAnalyzeCommandTarGZFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a .tar.gz file as the input
// THEN expect the command to output the correct summary
func TestAnalyzeCommandTarGZFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test.tar.gz")
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), istioIPErr)
}

// TestAnalyzeCommandTGZFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a .tgz file as the input
// THEN expect the command to output the correct summary
func TestAnalyzeCommandTGZFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test.tgz")
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), istioIPErr)
}

// TestAnalyzeCommandTarFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a .tar file as the input
// THEN expect the command to output the correct summary
func TestAnalyzeCommandTarFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test.tar")
	err := cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(buf), istioIPErr)
}

// TestAnalyzeCommandTarFileNotFound
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a non-existent path to a tar file
// THEN expect the command to return an error
func TestAnalyzeCommandTarFileNotFound(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test-bad-path.tgz")
	err := cmd.Execute()
	assert.ErrorContains(t, err, "an error occurred when trying to open ../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test-bad-path.tgz")
}

// TestAnalyzeCommandVZTarFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a tar.gz file that has been tarred using the CLI tool archive function
// THEN expect the command to not raise an error and output the correct summary
func TestAnalyzeCommandVZTarGZFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TarFileFlagName, "../../pkg/internal/test/cluster/tar-file-in-vz-format.tar.gz")
	err := cmd.Execute()
	assert.Nil(t, err)
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles(t *testing.T) (*os.File, *os.File) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	assert.NoError(t, err)

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	assert.NoError(t, err)

	return stdoutFile, stderrFile
}
