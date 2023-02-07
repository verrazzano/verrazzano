// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"testing"
)

const imagePullCase1 = "../../pkg/analysis/test/cluster/image-pull-case1/"
const ingressIPNotFound = "../../pkg/analysis/test/cluster/ingress-ip-not-found"

// TestAnalyzeCommandDetailedReport
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid capture-dir and report-format set to "detailed"
// THEN expect the command to provide the report containing all the details for one or more issues reported
func TestAnalyzeCommandDetailedReport(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.DetailedReport)
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.Contains(t, buf.String(), "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer",
		"Error syncing load balancer: failed to ensure load balancer: awaiting load balancer: context deadline exceeded")
}

// TestAnalyzeCommandSummaryReport
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid capture-dir and report-format set to "summary"
// THEN expect the command to provide the report containing only summary for one or more issues reported
func TestAnalyzeCommandSummaryReport(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.SummaryReport)
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.NotContains(t, buf.String(), "Error syncing load balancer: failed to ensure load balancer: awaiting load balancer: context deadline exceeded")
	assert.Contains(t, buf.String(), "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer")
}

// TestAnalyzeCommandInvalidReportFormat
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with an invalid value for report-format
// THEN expect the command to fail with an appropriate error message to indicate the issue
func TestAnalyzeCommandInvalidReportFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, "invalid-report-format")
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "\"invalid-report-format\" is not valid for flag report-format, only \"summary\" and \"detailed\" are valid")
}

// TestAnalyzeCommandDefaultReportFormat
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute without report-format
// THEN expect the command to take the default value of summary for report-format and perform the analysis
func TestAnalyzeCommandDefaultReportFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.NotContains(t, buf.String(), "Error syncing load balancer: failed to ensure load balancer: awaiting load balancer: context deadline exceeded")
	assert.Contains(t, buf.String(), "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer")
}

// TestAnalyzeCommandWithReportFile
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with a valid report-file
// THEN expect the command to create the report file, containing the analysis report
func TestAnalyzeCommandWithReportFile(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, constants.DetailedReport)
	cmd.PersistentFlags().Set(constants.ReportFileFlagName, "TestAnalyzeCommandReportFileOutput")
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.FileExists(t, "TestAnalyzeCommandReportFileOutput")
	os.Remove("TestAnalyzeCommandReportFileOutput")
	assert.NoFileExists(t, "TestAnalyzeCommandReportFileOutput")
}

// TestAnalyzeCommandInvalidCapturedDir
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute with capture-dir not containing the cluster snapshot
// THEN expect the command to fail with an appropriate error message
func TestAnalyzeCommandInvalidCapturedDir(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, "../")
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Cluster Analyzer runAnalysis didn't find any clusters")
}
