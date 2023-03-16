// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	installcmd "github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelper "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const imagePullCase1 = "../../pkg/analysis/test/cluster/image-pull-case1/"
const ingressIPNotFound = "../../pkg/analysis/test/cluster/ingress-ip-not-found"

const loadBalancerErr = "Error syncing load balancer: failed to ensure load balancer: awaiting load balancer: context deadline exceeded"
const noIPFoundErr = "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer"

// TestAnalyzeCommandDefault
// GIVEN a CLI analyze command
// WHEN I call cmd.Execute without specifying flag capture-dir
// THEN expect the command to analyze the live cluster
func TestAnalyzeCommandDefault(t *testing.T) {
	c := getClientWithWatch()
	installVZ(t, c)

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)

	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	rc.SetClient(c)
	cmd := NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	// This should generate a stdout from the live cluster
	assert.Contains(t, string(buf), "Verrazzano analysis CLI did not detect any issue in the cluster")
	// Clean analysis should not generate a report file
	fileMatched, _ := filepath.Glob(constants.VzAnalysisReportTmpFile)
	assert.Len(t, fileMatched, 0)
}

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

// getClientWithWatch returns a client for installing Verrazzano
func getClientWithWatch() client.WithWatch {
	vpo := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app":               constants.VerrazzanoPlatformOperator,
				"pod-template-hash": "45f78ffddd",
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
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	}
	replicaset := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      fmt.Sprintf("%s-45f78ffddd", constants.VerrazzanoPlatformOperator),
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(pkghelper.NewScheme()).WithObjects(vpo, deployment, replicaset).Build()
	return c
}

// installVZ installs Verrazzano using the given client
func installVZ(t *testing.T, c client.WithWatch) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
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
	buf, err := os.ReadFile(stderrFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(buf))
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles(t *testing.T) (*os.File, *os.File) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	assert.NoError(t, err)

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	assert.NoError(t, err)

	return stdoutFile, stderrFile
}
