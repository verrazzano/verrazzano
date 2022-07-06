// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"testing"
)

// TestBugReportHelp
// GIVEN a CLI bug-report command
//  WHEN I call cmd.Help for bug-report
//  THEN expect the help for the command in the standard output
func TestBugReportHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)
	err := cmd.Help()
	if err != nil {
		assert.Error(t, err)
	}
	assert.Contains(t, buf.String(), "Verrazzano command line utility to capture the data from the cluster, to report an issue")
}

// TestBugReportWithoutAnyFlag
// GIVEN a CLI bug-report command without mandatory flag --report-file
//  WHEN I call cmd.Execute for bug-report
//  THEN expect an error
func TestBugReportWithoutAnyFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Contains(t, err.Error(), "required flag(s) \"report-file\" not set")
}

// TestBugReportExistingReportFile
// GIVEN a CLI bug-report command using an existing file for flag --report-file
//  WHEN I call cmd.Execute for bug-report
//  THEN expect an error
func TestBugReportExistingReportFile(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	// Define and create the bug report file
	reportFile := "bug-report.tgz"
	bugRepFile, err := os.Create(tmpDir + string(os.PathSeparator) + reportFile)
	if err != nil {
		assert.Error(t, err)
	}
	defer bugRepFile.Close()

	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile.Name())
	err = cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("%s already exists", reportFile))
}
