// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	adminv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestUninstallCmd
// GIVEN a CLI uninstall command with all defaults and --wait==false
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command is successful
func TestUninstallCmd(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	validatingWebhookConfig := &adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperator,
		},
	}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperator,
		},
	}
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoManagedCluster,
		},
	}
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace, validatingWebhookConfig, clusterRoleBinding, clusterRole).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Run uninstall command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Uninstalling Verrazzano\n")
	assert.Contains(t, buf.String(), "Successfully uninstalled Verrazzano\n")

	// Expect the Verrazzano resource to be deleted
	v := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &v)
	assert.True(t, errors.IsNotFound(err))

	// Expect the install namespace to be deleted
	ns := corev1.Namespace{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoInstallNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Validating Webhook Configuration to be deleted
	vwc := adminv1.ValidatingWebhookConfiguration{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &vwc)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role Binding to be deleted
	crb := rbacv1.ClusterRoleBinding{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &crb)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role to be deleted
	cr := rbacv1.ClusterRole{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoManagedCluster}, &cr)
	assert.True(t, errors.IsNotFound(err))
}

// TestUninstallCmdUninstallJob
// GIVEN a CLI uninstall command with all defaults and --wait==false and a 1.3.1 version install
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command is successful
func TestUninstallCmdUninstallJob(t *testing.T) {
	deployment := createDeployment(nil)
	job := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoUninstall,
			Labels: map[string]string{
				"job-name": constants.VerrazzanoUninstall + "-verrazzano",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
	namespace := createNamespace()
	validatingWebhookConfig := &adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperator,
		},
	}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperator,
		},
	}
	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoManagedCluster,
		},
	}
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, job, vz, namespace, validatingWebhookConfig, clusterRoleBinding, clusterRole).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Run uninstall command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Uninstalling Verrazzano\n")
	assert.Contains(t, buf.String(), "Successfully uninstalled Verrazzano\n")

	// Expect the Verrazzano resource to be deleted
	v := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &v)
	assert.True(t, errors.IsNotFound(err))

	// Expect the install namespace to be deleted
	ns := corev1.Namespace{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoInstallNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Validating Webhook Configuration to be deleted
	vwc := adminv1.ValidatingWebhookConfiguration{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &vwc)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role Binding to be deleted
	crb := rbacv1.ClusterRoleBinding{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &crb)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role to be deleted
	cr := rbacv1.ClusterRole{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoManagedCluster}, &cr)
	assert.True(t, errors.IsNotFound(err))
}

// TestUninstallCmdDefaultTimeout
// GIVEN a CLI uninstall command with all defaults and --timeout=2ms
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command times out
func TestUninstallCmdDefaultTimeout(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	_ = cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")

	// Run upgrade command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	// This must be less than the 1 second polling delay to pass
	// since the Verrazzano resource gets deleted almost instantaneously
	assert.Contains(t, buf.String(), "Timeout 2ms exceeded waiting for uninstall to complete")
}

// TestUninstallCmdDefaultNoWait
// GIVEN a CLI uninstall command with all defaults and --wait==false
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command is successful
func TestUninstallCmdDefaultNoWait(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vz := createVz()
	vpo := createVpoPod()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vz, vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	_ = cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run uninstall command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestUninstallCmdJsonLogFormat
// GIVEN a CLI uninstall command with defaults and --log-format=json and --wait==false
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command is successful
func TestUninstallCmdJsonLogFormat(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vz := createVz()
	vpo := createVpoPod()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vz, vpo).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run uninstall command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestUninstallCmdDefaultNoVPO
// GIVEN a CLI uninstall command with all defaults and no VPO found
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command fails
func TestUninstallCmdDefaultNoVPO(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Failed to find the Verrazzano platform operator in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Failed to find the Verrazzano platform operator in namespace verrazzano-install")
}

// TestUninstallCmdDefaultNoUninstallJob
// GIVEN a CLI uninstall command with all defaults and no uninstall job pod
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command fails
func TestUninstallCmdDefaultNoUninstallJob(t *testing.T) {
	deployment := createDeployment(map[string]string{"app.kubernetes.io/version": "1.3.0"})
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "simple")

	setWaitRetries(1)
	defer resetWaitRetries()

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-uninstall-verrazzano, verrazzano-uninstall-verrazzano pod not found in namespace verrazzano-install")
	assert.Contains(t, errBuf.String(), "Error: Waiting for verrazzano-uninstall-verrazzano, verrazzano-uninstall-verrazzano pod not found in namespace verrazzano-install")
}

// TestUninstallCmdDefaultNoVzResource
// GIVEN a CLI uninstall command with all defaults and no vz resource found
//  WHEN I call cmd.Execute for uninstall
//  THEN the CLI uninstall command fails
func TestUninstallCmdDefaultNoVzResource(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Verrazzano is not installed: Failed to find any Verrazzano resources")
	assert.Contains(t, errBuf.String(), "Error: Verrazzano is not installed: Failed to find any Verrazzano resources")
}

func createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: vzconstants.VerrazzanoInstallNamespace,
		},
	}
}

func createVz() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
}

func createDeployment(labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels:    labels,
		},
	}
}

func createVpoPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
	}
}
