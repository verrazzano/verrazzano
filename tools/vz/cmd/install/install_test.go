// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestInstallCmdDefaultNoWait
// GIVEN a CLI install command with all defaults and --wait==false
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command is successful
func TestInstallCmdDefaultNoWait(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "webhook-init",
					Ready: true,
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "verrazzano-platform-operator",
					Ready: true,
				},
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestInstallCmdDefaultTimeout
// GIVEN a CLI install command with all defaults and --timeout=2s
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command times out
func TestInstallCmdDefaultTimeout(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2s")

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Timeout 2s exceeded waiting for install to complete")
}

// TestInstallCmdDefaultNoVPO
// GIVEN a CLI install command with all defaults and no VPO found
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command fails
func TestInstallCmdDefaultNoVPO(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)

	// Run install command
	vpoWaitRetries = 1 // override for unit testing
	err := cmd.Execute()
	resetVpoWaitRetries()
	assert.Error(t, err)
	assert.EqualError(t, err, "verrazzano-platform-operator pod not found in namespace verrazzano-install")
	assert.Equal(t, errBuf.String(), "Error: verrazzano-platform-operator pod not found in namespace verrazzano-install\n")
}

// TestInstallCmdDefaultMultipleVPO
// GIVEN a CLI install command with all defaults and multiple VPOs found
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command fails
func TestInstallCmdDefaultMultipleVPO(t *testing.T) {
	vpo1 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator + "-1",
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	vpo2 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator + "-2",
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo1, vpo2).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)

	// Run install command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.EqualError(t, err, "More than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	assert.Equal(t, errBuf.String(), "Error: More than one verrazzano-platform-operator pod was found in namespace verrazzano-install\n")
}

// TestInstallCmdJsonLogFormat
// GIVEN a CLI install command with defaults and --log-format=json and --wait==false
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command is successful
func TestInstallCmdJsonLogFormat(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestInstallCmdFilenames
// GIVEN a CLI install command with defaults and --wait=false and --filename specified
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command is successful
func TestInstallCmdFilenames(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.FilenameFlag, "../../test/testdata/dev-profile.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestInstallCmdOperatorFile
// GIVEN a CLI install command with defaults and --wait=false and --operator-file specified
//  WHEN I call cmd.Execute for install
//  THEN the CLI install command is successful
func TestInstallCmdOperatorFile(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      verrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": verrazzanoPlatformOperator,
			},
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "../../test/testdata/operator-file-fake.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

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
}

// TestInstallValidations
// GIVEN an install command
//  WHEN invalid command options exist
//  THEN expect an error
func TestInstallValidations(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "test")
	cmd.PersistentFlags().Set(constants.VersionFlag, "test")
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("--%s and --%s cannot both be specified", constants.VersionFlag, constants.OperatorFileFlag))
}

// TestGetWaitTimeoutDefault
// GIVEN no wait and timeout arguments specified
//  WHEN I call GetWaitTimeout
//  THEN the default timeout duration is returned
func TestGetWaitTimeoutDefault(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "30m0s", duration.String())
}

// TestGetWaitTimeoutNoWait
// GIVEN wait is specified as false
//  WHEN I call GetWaitTimeout
//  THEN the duration returned is zero
func TestGetWaitTimeoutNoWait(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "0s", duration.String())
}

// TestGetWaitTimeoutSpecified
// GIVEN wait the timeout is specified as 10m
//  WHEN I call GetWaitTimeout
//  THEN the duration returned is 10m0s
func TestGetWaitTimeoutSpecified(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "10m")
	duration, err := cmdHelpers.GetWaitTimeout(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "10m0s", duration.String())
}

// TestGetLogFormatSimple
// GIVEN simple log format argument specified
//  WHEN I call GetLogFormat
//  THEN the simple log format is returned
func TestGetLogFormatSimple(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "simple")
	logFormat, err := cmdHelpers.GetLogFormat(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "simple", logFormat.String())
}

// TestGetLogFormatJson
// GIVEN json log format is specified
//  WHEN I call GetLogFormat
//  THEN json log format is returned
func TestGetLogFormatJson(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdInstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	logFormat, err := cmdHelpers.GetLogFormat(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "json", logFormat.String())
}
