// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bytes"
	"context"
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/analyze"
	appsv1 "k8s.io/api/apps/v1"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	testKubeConfig   = "kubeconfig"
	testK8sContext   = "testcontext"
	testFilenamePath = "../../test/testdata/v1beta1.yaml"
)

// TestInstallCmdDefaultNoWait
// GIVEN a CLI install command with all defaults and --wait==false
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdDefaultNoWait(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)
}

// TestInstallCmdDefaultTimeoutBugReport
// GIVEN a CLI install command with all defaults and --timeout=2ms
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command times out and a bug report is generated
func TestInstallCmdDefaultTimeoutBugReport(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, buf, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, testFilenamePath)
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	defer os.RemoveAll(tempKubeConfigPath.Name())

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for install to complete\n", errBuf.String())
	assert.Contains(t, buf.String(), "Installing Verrazzano version v1.3.1")
	if !helpers.CheckBugReportExistsInDir("") {
		t.Fatal("cannot find bug report file in current directory")
	}
}

// TestInstallCmdDefaultTimeoutNoBugReport
// GIVEN a CLI install command with --timeout=2ms and auto-bug-report=false
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command times out and a bug report is not generated
func TestInstallCmdDefaultTimeoutNoBugReport(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, buf, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	cmd.PersistentFlags().Set(constants.FilenameFlag, testFilenamePath)
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmd.PersistentFlags().Set(constants.AutoBugReportFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer os.RemoveAll(tempKubeConfigPath.Name())
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for install to complete\n", errBuf.String())
	assert.Contains(t, buf.String(), "Installing Verrazzano version v1.3.1")
	// Bug report must not exist
	if helpers.CheckBugReportExistsInDir("") {
		t.Fatal("found bug report file in current directory")
	}
}

// TestInstallCmdDefaultNoVPO
// GIVEN a CLI install command with all defaults and no VPO found
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command fails
func TestInstallCmdDefaultNoVPO(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)

	// Run install command
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	cmd.PersistentFlags().Set(constants.VPOTimeoutFlag, "1s")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
}

// TestInstallCmdDefaultMultipleVPO
// GIVEN a CLI install command with all defaults and multiple VPOs found
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command fails
func TestInstallCmdDefaultMultipleVPO(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), testhelpers.CreateVPOPod(constants.VerrazzanoPlatformOperator+"-2"))...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)

	// Run install command
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	cmd.PersistentFlags().Set(constants.VPOTimeoutFlag, "1s")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
}

// TestInstallCmdJsonLogFormat
// GIVEN a CLI install command with defaults and --log-format=json and --wait==false
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdJsonLogFormat(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)
}

// TestInstallCmdMultipleGroupVersions
// GIVEN a CLI install command with defaults and --wait=false and --filename specified and multiple group versions in the filenames
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is unsuccessful
func TestInstallCmdMultipleGroupVersions(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/dev-profile.yaml")
	cmd.PersistentFlags().Set(constants.FilenameFlag, testFilenamePath)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot merge objects with different group versions")
}

func TestInstallCmdFilenamesV1Beta1(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, testFilenamePath)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "my-verrazzano"}, &vz)
	assert.NoError(t, err)
	assert.Equal(t, v1beta1.Dev, vz.Spec.Profile)
	assert.NotNil(t, vz.Spec.Components.IngressNGINX)
	assert.NotNil(t, vz.Spec.Components.Fluentd)
	assert.Equal(t, vz.Spec.Components.Fluentd.OpenSearchURL, "https://opensearch.com:9200/")
	assert.Equal(t, vz.Spec.Components.Fluentd.OpenSearchSecret, "foo")
}

// TestInstallCmdFilenames
// GIVEN a CLI install command with defaults and --wait=false and --filename specified
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdFilenames(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/dev-profile.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "my-verrazzano"}, &vz)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
}

// TestInstallCmdFilenamesCsv
// GIVEN a CLI install command with defaults and --wait=false and --filename specified as a comma separated list
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdFilenamesCsv(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/dev-profile.yaml,../../test/testdata/override-components.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "my-verrazzano"}, &vz)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
	assert.False(t, *vz.Spec.Components.Rancher.Enabled)
}

// TestInstallCmdSets
// GIVEN a CLI install command with defaults and --wait=false and --set specified
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdSets(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=dev")
	cmd.PersistentFlags().Set(constants.SetFlag, "environmentName=test")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
	assert.Equal(t, "test", vz.Spec.EnvironmentName)
}

// TestInstallCmdFilenamesAndSets
// GIVEN a CLI install command with defaults and --wait=false and --filename and --set specified
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdFilenamesAndSets(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/dev-profile.yaml")
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=prod")
	cmd.PersistentFlags().Set(constants.SetFlag, "environmentName=test")
	cmd.PersistentFlags().Set(constants.SetFlag, "components.ingress.overrides[0].values.controller.podLabels.override=\"true\"")
	cmd.PersistentFlags().Set(constants.SetFlag, "components.ingress.overrides[1].values.controller.service.annotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\"=flexible")
	cmd.PersistentFlags().Set(constants.SetFlag, "components.ingress.enabled=true")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	// Verify the vz resource is as expected
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "my-verrazzano"}, &vz)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.Prod, vz.Spec.Profile)
	assert.Equal(t, "test", vz.Spec.EnvironmentName)
	assert.Equal(t, true, *vz.Spec.Components.Ingress.Enabled)
	json, err := vz.Spec.Components.Ingress.InstallOverrides.ValueOverrides[0].Values.MarshalJSON()
	assert.NoError(t, err)
	outyaml, err := yaml.JSONToYAML(json)
	assert.NoError(t, err)
	assert.Equal(t, "controller:\n  podLabels:\n    override: \"true\"\n", string(outyaml))
	json, err = vz.Spec.Components.Ingress.InstallOverrides.ValueOverrides[1].Values.MarshalJSON()
	assert.NoError(t, err)
	outyaml, err = yaml.JSONToYAML(json)
	assert.NoError(t, err)
	assert.Equal(t, "controller:\n  service:\n    annotations:\n      service.beta.kubernetes.io/oci-load-balancer-shape: flexible\n", string(outyaml))
}

// TestInstallCmdOperatorFile
// GIVEN a CLI install command with defaults and --wait=false and --operator-file specified
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdOperatorFile(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, buf, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "../../test/testdata/operator-file-fake.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Applying the file ../../test/testdata/operator-file-fake.yaml\nnamespace/verrazzano-install created\nserviceaccount/verrazzano-platform-operator created\nservice/verrazzano-platform-operator created\n")

	// Verify the objects in the operator-file got added
	sa := corev1.ServiceAccount{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "verrazzano-install", Name: "verrazzano-platform-operator"}, &sa)
	assert.NoError(t, err)

	ns := corev1.Namespace{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-install"}, &ns)
	assert.NoError(t, err)

	svc := corev1.Service{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "verrazzano-install", Name: "verrazzano-platform-operator"}, &svc)
	assert.NoError(t, err)

	// Verify the vz resource is as expected
	vz := v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)
}

// TestInstallValidations
// GIVEN an install command
//
//	WHEN invalid command options exist
//	THEN expect an error
func TestInstallValidations(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "test")
	cmd.PersistentFlags().Set(constants.VersionFlag, "test")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("--%s and --%s cannot both be specified", constants.VersionFlag, constants.OperatorFileFlag))
}

// TestGetWaitTimeoutDefault
// GIVEN no wait and timeout arguments specified
//
//	WHEN I call GetWaitTimeout
//	THEN the default timeout duration is returned
func TestGetWaitTimeoutDefault(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	duration, err := cmdHelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	assert.NoError(t, err)
	assert.Equal(t, "30m0s", duration.String())
}

// TestGetWaitTimeoutNoWait
// GIVEN wait is specified as false
//
//	WHEN I call GetWaitTimeout
//	THEN the duration returned is zero
func TestGetWaitTimeoutNoWait(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	duration, err := cmdHelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	assert.NoError(t, err)
	assert.Equal(t, "0s", duration.String())
}

// TestGetWaitTimeoutSpecified
// GIVEN wait the timeout is specified as 10m
//
//	WHEN I call GetWaitTimeout
//	THEN the duration returned is 10m0s
func TestGetWaitTimeoutSpecified(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "10m")
	duration, err := cmdHelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	assert.NoError(t, err)
	assert.Equal(t, "10m0s", duration.String())
}

// TestGetLogFormatSimple
// GIVEN simple log format argument specified
//
//	WHEN I call GetLogFormat
//	THEN the simple log format is returned
func TestGetLogFormatSimple(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "simple")
	logFormat, err := cmdHelpers.GetLogFormat(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "simple", logFormat.String())
}

// TestGetLogFormatJson
// GIVEN json log format is specified
//
//	WHEN I call GetLogFormat
//	THEN json log format is returned
func TestGetLogFormatJson(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	logFormat, err := cmdHelpers.GetLogFormat(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "json", logFormat.String())
}

// TestSetCommandInvalidFormat
// GIVEN a set command is specified with the invalid format
//
//	WHEN I call getSetArguments
//	THEN an error is returned
func TestSetCommandInvalidFormat(t *testing.T) {
	cmd, _, errBuf, rc := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.SetFlag, "badflag")
	propValues, err := getSetArguments(cmd, rc)
	assert.Nil(t, propValues)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "Invalid set flag(s) specified")
	assert.Equal(t, "Invalid set flag \"badflag\" specified. Flag must be specified in the format path=value\n", errBuf.String())
}

// TestSetCommandSingle
// GIVEN a single set command
//
//	WHEN I call getSetArguments
//	THEN the expected property value is returned
func TestSetCommandSingle(t *testing.T) {
	cmd, _, _, rc := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=dev")
	propValues, err := getSetArguments(cmd, rc)
	assert.NoError(t, err)
	assert.Len(t, propValues, 1)
	assert.Contains(t, propValues["spec.profile"], "dev")
}

// TestSetCommandMultiple
// GIVEN multiple set commands
//
//	WHEN I call getSetArguments
//	THEN the expected property values are returned
func TestSetCommandMultiple(t *testing.T) {
	cmd, _, _, rc := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=dev")
	cmd.PersistentFlags().Set(constants.SetFlag, "spec.environmentName=default")
	propValues, err := getSetArguments(cmd, rc)
	assert.NoError(t, err)
	assert.Len(t, propValues, 2)
	assert.Contains(t, propValues["spec.profile"], "dev")
	assert.Contains(t, propValues["spec.environmentName"], "default")
}

// TestSetCommandOverride
// GIVEN multiple set commands overriding the same property
//
//	WHEN I call getSetArguments
//	THEN the expected property values are returned
func TestSetCommandOverride(t *testing.T) {
	cmd, _, _, rc := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=dev")
	cmd.PersistentFlags().Set(constants.SetFlag, "profile=prod")
	propValues, err := getSetArguments(cmd, rc)
	assert.NoError(t, err)
	assert.Len(t, propValues, 1)
	assert.Contains(t, propValues["spec.profile"], "prod")
}

// TestInstallCmdInProgress
// GIVEN a CLI install command when an install was in progress
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallCmdInProgress(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: v1beta1.VerrazzanoStatus{
			State:   v1beta1.VzStateReconciling,
			Version: "v1.3.1",
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestInstallCmdAlreadyInstalled
// GIVEN a CLI install command when an install already happened
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is unsuccessful
func TestInstallCmdAlreadyInstalled(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: v1beta1.VerrazzanoStatus{
			State:   v1beta1.VzStateReady,
			Version: "v1.3.1",
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Only one install of Verrazzano is allowed")
}

// TestInstallCmdDifferentVersion
// GIVEN a CLI install command when an install is in progress for a different version
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is unsuccessful
func TestInstallCmdDifferentVersion(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: v1beta1.VerrazzanoStatus{
			State:   v1beta1.VzStateReconciling,
			Version: "v1.3.2",
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to install version v1.3.1, install of version v1.3.2 is in progress")
}

func createNewTestCommandAndBuffers(t *testing.T, c client.Client) (*cobra.Command, *bytes.Buffer, *bytes.Buffer, *testhelpers.FakeRootCmdContext) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	if c != nil {
		rc.SetClient(c)
	}
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	return cmd, buf, errBuf, rc
}

// installVZ installs Verrazzano using the given client
func installVZ(t *testing.T, c client.WithWatch) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
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
		os.Remove(bugReportFilePath)
	}()
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	rc.SetClient(c)
	cmd := analyze.NewCmdAnalyze(rc)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
	buf, err := os.ReadFile(stdoutFile.Name())
	assert.NoError(t, err)
	// This should generate a stdout from the live cluster
	assert.Contains(t, string(buf), "Verrazzano analysis CLI did not detect any issue in")
	// Clean analysis should not generate a report file
	fileMatched, _ := filepath.Glob(constants.VzAnalysisReportTmpFile)
	assert.Len(t, fileMatched, 0)
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
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo, deployment, replicaset).Build()
	return c
}
