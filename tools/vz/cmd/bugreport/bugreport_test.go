// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	installcmd "github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelper "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	testhelper "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
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
)

// TestBugReportHelp
// GIVEN a CLI bug-report command
//
//	WHEN I call cmd.Help for bug-report
//	THEN expect the help for the command in the standard output
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
	assert.Contains(t, buf.String(), "Verrazzano command line utility to collect data from the cluster, to report an issue")
}

// TestBugReportExistingReportFile
// GIVEN a CLI bug-report command using an existing file for flag --report-file
//
//	WHEN I call cmd.Execute for bug-report
//	THEN expect an error
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
//
//	WHEN I call cmd.Execute for bug-report
//	THEN expect an error
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
//
//	WHEN I call cmd.Execute for bug-report
//	THEN expect an error
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
//
//	WHEN I call cmd.Execute for bug-report
//	THEN expect an error
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
//
//	WHEN I call cmd.Execute
//	THEN expect the command to show the resources captured in the standard output and create the bug report file
func TestBugReportSuccess(t *testing.T) {
	c := getClientWithWatch()
	installVZ(t, c)

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
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
	cmd.PersistentFlags().Set(constants.VerboseFlag, "true")
	err = cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Capturing  resources from the cluster", "Capturing Verrazzano resource",
		"Capturing log from pod verrazzano-platform-operator in verrazzano-install namespace",
		"Successfully created the bug report",
		"WARNING: Please examine the contents of the bug report for sensitive data", "Namespace dummy not found in the cluster")
	assert.FileExists(t, bugRepFile)

	// Validate the fact that --verbose is disabled by default
	buf = new(bytes.Buffer)
	errBuf = new(bytes.Buffer)
	rc = helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	bugRepFile = tmpDir + string(os.PathSeparator) + "bug-report-verbose-false.tgz"
	cmd = NewCmdBugReport(rc)
	cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	err = cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Capturing  resources from the cluster",
		"Successfully created the bug report",
		"WARNING: Please examine the contents of the bug report for sensitive data")
	assert.NotContains(t, buf.String(), "Capturing Verrazzano resource",
		"Capturing log from pod verrazzano-platform-operator in verrazzano-install namespace")
	assert.FileExists(t, bugRepFile)
}

// TestBugReportDefaultReportFile
// GIVEN a CLI bug-report command
//
//	WHEN I call cmd.Execute, without specifying --report-file
//	THEN expect the command to create the report bug-report.tar.gz under the current directory
func TestBugReportDefaultReportFile(t *testing.T) {
	c := getClientWithWatch()
	installVZ(t, c)

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	cmd.PersistentFlags().Set(constants.VerboseFlag, "true")
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Capturing Verrazzano resource",
		"Capturing log from pod verrazzano-platform-operator in verrazzano-install namespace",
		"Created the bug report",
		"WARNING: Please examine the contents of the bug report for sensitive data", "Namespace dummy not found in the cluster")
	currentDir, err := os.Getwd()
	if err != nil {
		assert.Error(t, err)
	}
	defaultBugReport := currentDir + string(os.PathSeparator) + constants.BugReportFileDefaultValue
	assert.FileExists(t, defaultBugReport)
	os.Remove(defaultBugReport)
}

// TestBugReportV1Alpha1Verrazzano
// GIVEN a CLI bug-report command
//
//	WHEN I call cmd.Execute with a v1alpha1 Verrazzano installed
//	THEN expect the command to resolve the v1alpha1 to a v1beta1 and return no error
func TestBugReportV1Alpha1Verrazzano(t *testing.T) {
	c := getClientWithV1Alpha1VZWatch()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := os.MkdirTemp("", "bug-report")

	bugRepFile := tmpDir + string(os.PathSeparator) + "bug-report-v1alpha1.tgz"
	err := cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	assert.NoError(t, err)
	err = cmd.PersistentFlags().Set(constants.BugReportIncludeNSFlagName, "dummy,verrazzano-install")
	assert.NoError(t, err)
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.FileExists(t, bugRepFile)
	os.Remove(bugRepFile)
}

// TestBugReportNoVerrazzano
// GIVEN a CLI bug-report command
//
//	WHEN I call cmd.Execute without Verrazzano installed
//	THEN expect the command to display a message indicating Verrazzano is not installed
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
	assert.Contains(t, errBuf.String(), "Verrazzano is not installed")
}

// TestBugReportFailureUsingInvalidClient
// GIVEN a CLI bug-report command
//
//	WHEN I call cmd.Execute without Verrazzano installed and using an invalid client
//	THEN expect the command to fail with a message indicating Verrazzano is not installed and no resource captured
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

	assert.Contains(t, errBuf.String(), "Verrazzano is not installed")
	assert.NoFileExists(t, bugRepFile)
}

// getClientWithWatch returns a client for installing Verrazzano
func getClientWithWatch() client.WithWatch {
	return fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(getVpoObjects()[1:]...).Build()
}

// getClientWithV1Alpha1VZWatch returns a client containing all VPO objects and the v1alpha1 Verrazzano CR
func getClientWithV1Alpha1VZWatch() client.WithWatch {
	return fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(getVpoObjects()[1:]...).
		WithObjects(&v1alpha1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
			Spec: v1alpha1.VerrazzanoSpec{
				Profile: v1alpha1.Dev,
			},
		}).Build()
}

func getVpoObjects() []client.Object {
	return []client.Object{
		&v1beta1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
			Spec: v1beta1.VerrazzanoSpec{
				Profile: v1beta1.Dev,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconstants.VerrazzanoInstallNamespace,
				Name:      constants.VerrazzanoPlatformOperator,
				Labels: map[string]string{
					"app":               constants.VerrazzanoPlatformOperator,
					"pod-template-hash": "45f78ffddd",
				},
			},
		},
		&appsv1.Deployment{
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
				AvailableReplicas: 1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconstants.VerrazzanoInstallNamespace,
				Name:      fmt.Sprintf("%s-45f78ffddd", constants.VerrazzanoPlatformOperator),
				Annotations: map[string]string{
					"deployment.kubernetes.io/revision": "1",
				},
			},
		},
	}
}

// getInvalidClient returns an invalid client
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
	}
	c := fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(testObj, deployment).Build()
	return c
}

// installVZ installs Verrazzano using the given client
func installVZ(t *testing.T, c client.WithWatch) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelper.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := installcmd.NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// getClientWithVZWatch returns a client containing all VPO objects and the Verrazzano CR
func getClientWithVZWatch() client.WithWatch {
	return fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(getVpoObjects()...).Build()
}

// cleanupTempDir cleans up the given temp directory after the test run
func cleanupTempDir(t *testing.T, dirName string) {
	if err := os.RemoveAll(dirName); err != nil {
		t.Fatalf("Remove directory failed: %v", err)
	}
}

// TestBugReportSuccess
// GIVEN a CLI bug-report command
// WHEN I call cmd.Execute with include logs of  additional namespace and duration
// THEN expect the command to show the resources captured in the standard output and create the bug report file
func TestBugReportSuccessWithDuration(t *testing.T) {
	c := getClientWithVZWatch()

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdBugReport(rc)
	assert.NotNil(t, cmd)

	tmpDir, _ := os.MkdirTemp("", "bug-report")
	defer cleanupTempDir(t, tmpDir)

	bugRepFile := tmpDir + string(os.PathSeparator) + "bug-report.tgz"
	assert.NoError(t, err)
	err = cmd.PersistentFlags().Set(constants.BugReportFileFlagName, bugRepFile)
	assert.NoError(t, err)
	err = cmd.PersistentFlags().Set(constants.BugReportIncludeNSFlagName, "dummy,verrazzano-install,default,test")
	assert.NoError(t, err)
	err = cmd.PersistentFlags().Set(constants.VerboseFlag, "true")
	assert.NoError(t, err)
	err = cmd.PersistentFlags().Set(constants.BugReportLogFlagName, "true")
	assert.NoError(t, err)
	// If invalid time value is given then error is expected
	err = cmd.PersistentFlags().Set(constants.BugReportTimeFlagName, "3t")
	assert.Error(t, err)
	// Valid time value
	err = cmd.PersistentFlags().Set(constants.BugReportTimeFlagName, "3h")
	assert.NoError(t, err)
	err = cmd.Execute()
	if err != nil {
		assert.Error(t, err)
	}

	assert.NoError(t, err)
}
