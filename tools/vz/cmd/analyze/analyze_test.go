// Copyright (c) 2022, Oracle and/or its affiliates.
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

func TestAnalyzeCommandDefault(t *testing.T) {
	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Contains(t, err.Error(), "required flag(s) \"capture-dir\" not set")
}

func TestAnalyzeCommandValidCapturedDir(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, "simple")
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.Contains(t, buf.String(), "no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer")
}

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
	assert.Contains(t, err.Error(), "unsupported output format")
}

func TestAnalyzeCommandValidReportFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, ingressIPNotFound)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, "simple")
	err := cmd.Execute()
	assert.Nil(t, err)
}

func TestAnalyzeCommandReportFileOutputSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.DirectoryFlagName, imagePullCase1)
	cmd.PersistentFlags().Set(constants.ReportFormatFlagName, "simple")
	cmd.PersistentFlags().Set(constants.ReportFileFlagName, "TestAnalyzeCommandReportFileOutput")
	err := cmd.Execute()
	assert.Nil(t, err)
	assert.FileExists(t, "TestAnalyzeCommandReportFileOutput")
	os.Remove("TestAnalyzeCommandReportFileOutput")
	assert.NoFileExists(t, "TestAnalyzeCommandReportFileOutput")
}

func TestAnalyzeCommandCapturedDirInvalid(t *testing.T) {
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
