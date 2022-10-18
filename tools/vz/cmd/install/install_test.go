// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bytes"
	"context"
	"fmt"
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
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
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

// TestInstallCmdDefaultTimeout
// GIVEN a CLI install command with all defaults and --timeout=2s
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command times out
func TestInstallCmdDefaultTimeout(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, buf, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2s")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Timeout 2s exceeded waiting for install to complete\n", errBuf.String())
	assert.Contains(t, buf.String(), "Installing Verrazzano version v1.3.1")
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
	cmdHelpers.SetVpoWaitRetries(1) // override for unit testing
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	err := cmd.Execute()
	cmdHelpers.ResetVpoWaitRetries()
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
	cmdHelpers.SetVpoWaitRetries(1) // override for unit testing
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	err := cmd.Execute()
	cmdHelpers.ResetVpoWaitRetries()
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
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/v1beta1.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot merge objects with different group versions")
}

func TestInstallCmdFilenamesV1Beta1(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, _, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/v1beta1.yaml")
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
	vz := v1alpha1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vz)
	assert.NoError(t, err)
}

// TestInstallValidations
// GIVEN an install command
//
//	WHEN invalid command options exist
//	THEN expect an error
func TestInstallValidations(t *testing.T) {
	cmd, _, _, _ := createNewTestCommandAndBuffers(t, nil)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "test")
	cmd.PersistentFlags().Set(constants.VersionFlag, "test")
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
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
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
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
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
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
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
