// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestUpgradeCmdDefaultNoWait
// GIVEN a CLI upgrade command with all defaults and --wait==false
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdDefaultNoWait(t *testing.T) {
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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo, deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo, deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2s")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Timeout 2s exceeded waiting for upgrade to complete\n", errBuf.String())
	assert.Contains(t, buf.String(), "Upgrading Verrazzano to version v1.4.0")
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
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

	// Run upgrade command
	cmdHelpers.SetVpoWaitRetries(1) // override for unit testing
	err := cmd.Execute()
	cmdHelpers.ResetVpoWaitRetries()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
}

// TestUpgradeCmdDefaultMultipleVPO
// GIVEN a CLI upgrade command with all defaults and multiple VPOs found
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdDefaultMultipleVPO(t *testing.T) {
	vpo1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app":               constants.VerrazzanoPlatformOperator,
				"pod-template-hash": "56f78ffcfd",
			},
		},
	}
	vpo2 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator + "-2",
			Labels: map[string]string{
				"app":               constants.VerrazzanoPlatformOperator,
				"pod-template-hash": "56f78ffcfe",
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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo1, vpo2, deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

	// Run upgrade command
	cmdHelpers.SetVpoWaitRetries(1) // override for unit testing
	err := cmd.Execute()
	cmdHelpers.ResetVpoWaitRetries()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
}

// TestUpgradeCmdJsonLogFormat
// GIVEN a CLI upgrade command with defaults and --log-format=json and --wait==false
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdJsonLogFormat(t *testing.T) {
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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo, deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

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
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vpo, deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.OperatorFileFlag, "../../test/testdata/operator-file-fake.yaml")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")

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

	// Verify the version got updated
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, vz)
	assert.NoError(t, err)
	assert.Equal(t, "v1.4.0", vz.Spec.Version)
}

// TestUpgradeCmdNoVerrazzano
// GIVEN a CLI upgrade command with no verrazzano install resource found
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdNoVerrazzano(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects().Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Verrazzano is not installed: Failed to find any Verrazzano resources\n", errBuf.String())
}

// TestUpgradeCmdLesserVersion
// GIVEN a CLI upgrade command specifying a version less than the installed version
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command fails
func TestUpgradeCmdLesserVersion(t *testing.T) {
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Status: vzapi.VerrazzanoStatus{Version: "v1.3.4"},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.3.3")

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Upgrade to a lesser version of Verrazzano is not allowed. Upgrade version specified was v1.3.3 and current Verrazzano version is v1.3.4\n", errBuf.String())
}
