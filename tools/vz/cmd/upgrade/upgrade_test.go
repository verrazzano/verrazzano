// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestUpgradeCmdDefaultNoWait
// GIVEN a CLI upgrade command with all defaults and --wait==false
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdDefaultNoWait(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run upgrade command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestUpgradeCmdDefaultTimeout
// GIVEN a CLI upgrade command with all defaults and --timeout=2s
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command times out
func TestUpgradeCmdDefaultTimeout(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2s")

	// Run upgrade command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Timeout 2s exceeded waiting for upgrade to complete")
}

// TestUpgradeCmdDefaultNoVPO
// GIVEN a CLI upgrade command with all defaults and no VPO found
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdDefaultNoVPO(t *testing.T) {
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)

	// Run upgrade command
	cmdHelpers.SetVpoWaitRetries(1) // override for unit testing
	err := cmd.Execute()
	cmdHelpers.ResetVpoWaitRetries()
	assert.Error(t, err)
	assert.EqualError(t, err, "verrazzano-platform-operator pod not found in namespace verrazzano-install")
	assert.Equal(t, errBuf.String(), "Error: verrazzano-platform-operator pod not found in namespace verrazzano-install\n")
}

// TestUpgradeCmdDefaultMultipleVPO
// GIVEN a CLI upgrade command with all defaults and multiple VPOs found
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdDefaultMultipleVPO(t *testing.T) {
	vpo1 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator + "-1",
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
	vpo2 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator + "-2",
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo1, vpo2, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.EqualError(t, err, "More than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	assert.Equal(t, errBuf.String(), "Error: More than one verrazzano-platform-operator pod was found in namespace verrazzano-install\n")
}

// TestUpgradeCmdJsonLogFormat
// GIVEN a CLI upgrade command with defaults and --log-format=json and --wait==false
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdJsonLogFormat(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run upgrade command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestUpgradeCmdOperatorFile
// GIVEN a CLI upgrade command with defaults and --wait=false and --operator-file specified
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdOperatorFile(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "../../test/testdata/operator-file-fake.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run upgrade command
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

// TestUpgradeCmdNoVerrazzano
// GIVEN a CLI upgrade command with no verrazzano install resource found
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdNoVerrazzano(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
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
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Verrazzano is not installed: Failed to find any Verrazzano resources\n", errBuf.String())
}

// TestUpgradeValidations
// GIVEN an upgrade command
//  WHEN invalid command options exist
//  THEN expect an error
func TestUpgradeValidations(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "test")
	cmd.PersistentFlags().Set(constants.VersionFlag, "test")
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("--%s and --%s cannot both be specified", constants.VersionFlag, constants.OperatorFileFlag))
}
