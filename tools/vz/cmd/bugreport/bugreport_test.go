// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelper "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
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

// TestBugReportExistingDir
// GIVEN a CLI bug-report command with flag --report-file pointing to an existing directory
//  WHEN I call cmd.Execute for bug-report
//  THEN expect an error
func TestBugReportExistingDir(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	reportDir := tmpDir + string(os.PathSeparator) + "test-report"
	if err := os.Mkdir(reportDir, os.ModePerm); err != nil {
		assert.Error(t, err)
	}

	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, reportDir)
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test-report is an existing directory")
}

// TestBugReportNonExistingFileDir
// GIVEN a CLI bug-report command with flag --report-file pointing to a file, where the directory doesn't exist
//  WHEN I call cmd.Execute for bug-report
//  THEN expect an error
func TestBugReportNonExistingFileDir(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	reportDir := tmpDir + string(os.PathSeparator) + "test-report"
	reportFile := reportDir + string(os.PathSeparator) + string(os.PathSeparator) + "bug-report.tgz"

	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, reportFile)
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test-report: no such file or directory")
}

// TestBugReportFileNoPermission
// GIVEN a CLI bug-report command with flag --report-file pointing to a file, where there is no write permission
//  WHEN I call cmd.Execute for bug-report
//  THEN expect an error
func TestBugReportFileNoPermission(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	reportDir := tmpDir + string(os.PathSeparator) + "test-report"
	// Create a directory with only read permission
	if err := os.Mkdir(reportDir, 0400); err != nil {
		assert.Error(t, err)
	}
	reportFile := reportDir + string(os.PathSeparator) + "bug-report.tgz"
	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, reportFile)
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "permission denied to create the bug report")
}

// TestBugReportSuccess
// GIVEN a CLI bug-report command
//  WHEN I call cmd.Execute
//  THEN expect the command to show the resources captured in the standard output and create the bug report file
func TestBugReportSuccess(t *testing.T) {
	c := getClientWithWatch()
	installVZ(t, c)

	// Verify the vz resource is as expected
	vz := vzapi.Verrazzano{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	bugRepFile := tmpDir + string(os.PathSeparator) + "bug-report.tgz"
	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	cmd.PersistentFlags().Set(constants.BugReportIncludeNSFlagName, "dummy,verrazzano-install,default")
	err = cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Capturing Verrazzano resource",
		"Capturing log from pod verrazzano-platform-operator in verrazzano-install namespace",
		"Successfully created the bug report",
		"WARNING: Please examine the contents of the bug report for sensitive data", "Namespace dummy not found in the cluster")
	assert.FileExists(t, bugRepFile)
}

// TestBugReportNoVerrazzano
// GIVEN a CLI bug-report command
//  WHEN I call cmd.Execute without Verrazzano installed
//  THEN expect the command to display a message indicating Verrazzano is not installed
func TestBugReportNoVerrazzano(t *testing.T) {
	c := getClientWithWatch()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	bugRepFile := tmpDir + string(os.PathSeparator) + "bug-report.tgz"
	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	cmd.PersistentFlags().Set(constants.BugReportIncludeNSFlagName, "dummy,verrazzano-install")
	err := cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Verrazzano is not installed")
}

// TestBugReportFailureUsingInvalidClient
// GIVEN a CLI bug-report command
//  WHEN I call cmd.Execute without Verrazzano installed and using an invalid client
//  THEN expect the command to fail with a message indicating Verrazzano is not installed and no resource captured
func TestBugReportFailureUsingInvalidClient(t *testing.T) {
	c := getInvalidClient()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := ioutil.TempDir("", "bug-report")
	defer os.RemoveAll(tmpDir)

	bugRepFile := tmpDir + string(os.PathSeparator) + "bug-report.tgz"
	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	cmd.PersistentFlags().Set(constants.BugReportIncludeNSFlagName, "dummy,verrazzano-install")
	err := cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}
	assert.Contains(t, buf.String(), "Verrazzano is not installed")
	assert.Contains(t, errBuf.String(), "The bug-report command did not collect any file from the cluster")
	assert.NoFileExists(t, bugRepFile)
}

func getClientWithWatch() client.WithWatch {
	vpo := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app":               constants.VerrazzanoPlatformOperator,
				"pod-template-hash": "56f78ffcfd",
			},
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": constants.VerrazzanoPlatformOperator},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:               appsv1.DeploymentAvailable,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now().Add(time.Minute * 10)),
				},
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(vpo, deployment).Build()
	return c
}

func getInvalidClient() client.WithWatch {
	testObj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name:      "testpod",
			Labels: map[string]string{
				"app":               "test-app",
				"pod-template-hash": "56f78ddcfd",
			},
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testnamespace",
			Name:      "testpod",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-app"},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:               appsv1.DeploymentAvailable,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now().Add(time.Minute * 10)),
				},
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(testObj, deployment).Build()
	return c
}

func installVZ(t *testing.T, c client.WithWatch) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := install.NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}
