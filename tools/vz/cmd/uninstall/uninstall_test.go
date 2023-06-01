// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testKubeConfig    = "kubeconfig"
	testK8sContext    = "testcontext"
	VzVpoFailureError = "Failed to find the Verrazzano platform operator in namespace verrazzano-install"
	PodNotFoundError  = "Waiting for verrazzano-uninstall-verrazzano, verrazzano-uninstall-verrazzano pod not found in namespace verrazzano-install"
	BugReportNotExist = "cannot find bug report file in current directory"
)

// TestUninstallCmd
// GIVEN a CLI uninstall command with all defaults
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command is successful
func TestUninstallCmd(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	validatingWebhookConfig := createWebhook()
	clusterRoleBinding := createClusterRoleBinding()
	mcClusterRole := createMCClusterRole()
	registrarClusterRole := createRegistrarClusterRoleForRancher()
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace, validatingWebhookConfig, clusterRoleBinding, mcClusterRole, registrarClusterRole).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Uninstalling Verrazzano\n")

	// Ensure resources have been deleted
	ensureResourcesDeleted(t, c)
}

// TestUninstallCmdUninstallJob
// GIVEN a CLI uninstall command with all defaults and a 1.3.1 version install
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command is successful
func TestUninstallCmdUninstallJob(t *testing.T) {
	deployment := createVpoDeployment(nil)
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
	validatingWebhookConfig := createWebhook()
	clusterRoleBinding := createClusterRoleBinding()
	mcClusterRole := createMCClusterRole()
	registrarClusterRole := createRegistrarClusterRoleForRancher()
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, job, vz, namespace, validatingWebhookConfig, clusterRoleBinding, mcClusterRole, registrarClusterRole).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
	assert.Contains(t, buf.String(), "Uninstalling Verrazzano\n")

	// Ensure resources have been deleted
	ensureResourcesDeleted(t, c)
}

// TestUninstallCmdDefaultTimeout
// GIVEN a CLI uninstall command with all defaults and --timeout=2ms
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command times out and a bug-report is generated
func TestUninstallCmdDefaultTimeout(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	vz := createVz()
	webhook := createWebhook()
	mcClusterRole := createMCClusterRole()
	registrarClusterRole := createRegistrarClusterRoleForRancher()
	crb := createClusterRoleBinding()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace, webhook, mcClusterRole, registrarClusterRole, crb).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	_ = cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	defer os.RemoveAll(tempKubeConfigPath.Name())

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	// This must be less than the 1 second polling delay to pass
	// since the Verrazzano resource gets deleted almost instantaneously
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for uninstall to complete\n", errBuf.String())
	ensureResourcesNotDeleted(t, c)
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal(BugReportNotExist)
	}
}

// TestUninstallCmdDefaultTimeoutNoBugReport
// GIVEN a CLI uninstall command with all defaults, --timeout=2ms, and auto-bug-report=false
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command times out and bug-report is not generated
func TestUninstallCmdDefaultTimeoutNoBugReport(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	vz := createVz()
	webhook := createWebhook()
	mcClusterRole := createMCClusterRole()
	registrarClusterRole := createRegistrarClusterRoleForRancher()
	crb := createClusterRoleBinding()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace, webhook, mcClusterRole, registrarClusterRole, crb).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	_ = cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	_ = cmd.PersistentFlags().Set(constants.AutoBugReportFlag, "false")

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run upgrade command
	err := cmd.Execute()
	assert.Error(t, err)
	// This must be less than the 1 second polling delay to pass
	// since the Verrazzano resource gets deleted almost instantaneously
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for uninstall to complete\n", errBuf.String())
	ensureResourcesNotDeleted(t, c)
	// Bug Report must not exist
	if helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal("found bug report file in current directory")
	}
}

// TestUninstallCmdDefaultNoWait
// GIVEN a CLI uninstall command with all defaults and --wait==false
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command is successful
func TestUninstallCmdDefaultNoWait(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vpo := createVpoPod()
	namespace := createNamespace()
	vz := createVz()
	webhook := createWebhook()
	mcClusterRole := createMCClusterRole()
	registrarClusterRole := createRegistrarClusterRoleForRancher()
	crb := createClusterRoleBinding()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vpo, vz, namespace, webhook, mcClusterRole, registrarClusterRole, crb).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	_ = cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())

	ensureResourcesNotDeleted(t, c)
}

// TestUninstallCmdJsonLogFormat
// GIVEN a CLI uninstall command with defaults and --log-format=json and --wait==false
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command is successful
func TestUninstallCmdJsonLogFormat(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
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

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}

// TestUninstallCmdDefaultNoVPO
// GIVEN a CLI uninstall command with all defaults and no VPO found
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command fails and a bug-report is generated
func TestUninstallCmdDefaultNoVPO(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
	vz := createVz()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(deployment, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	assert.NotNil(t, cmd)
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	defer os.RemoveAll(tempKubeConfigPath.Name())

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, VzVpoFailureError)
	assert.Contains(t, errBuf.String(), VzVpoFailureError)
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal(BugReportNotExist)
	}
}

// TestUninstallCmdDefaultNoUninstallJob
// GIVEN a CLI uninstall command with all defaults and no uninstall job pod
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command fails and a bug-report is generated
func TestUninstallCmdDefaultNoUninstallJob(t *testing.T) {
	deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.3.0"})
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
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")

	setWaitRetries(1)
	defer resetWaitRetries()
	defer os.RemoveAll(tempKubeConfigPath.Name())

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, PodNotFoundError)
	assert.Contains(t, errBuf.String(), PodNotFoundError)
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal(BugReportNotExist)
	}
}

// TestUninstallCmdDefaultNoVzResource
// GIVEN a CLI uninstall command with all defaults and no vz resource found
//
//	WHEN I call cmd.Execute for uninstall
//	THEN the CLI uninstall command fails and bug report should not be generated
func TestUninstallCmdDefaultNoVzResource(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUninstall(rc)
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	assert.NotNil(t, cmd)

	// Suppressing uninstall prompt
	cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")

	// Run uninstall command
	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Verrazzano is not installed: Failed to find any Verrazzano resources")
	assert.Contains(t, errBuf.String(), "Verrazzano is not installed: Failed to find any Verrazzano resources")
	if helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal(BugReportNotExist)
	}
}

// TestUninstallWithConfirmUninstallFlag
// Given the "--skip-confirmation or -y" flag the Verrazzano Uninstall prompt will be suppressed
// any other input to the command-line other than Y or y will kill the uninstall process
func TestUninstallWithConfirmUninstallFlag(t *testing.T) {
	type fields struct {
		cmdLineInput  string
		doesUninstall bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"Suppress Uninstall prompt with --skip-confirmation=true", fields{cmdLineInput: "", doesUninstall: true}},
		{"Proceed with Uninstall, Y", fields{cmdLineInput: "Y", doesUninstall: true}},
		{"Proceed with Uninstall, y", fields{cmdLineInput: "y", doesUninstall: true}},
		{"Halt with Uninstall, n", fields{cmdLineInput: "n", doesUninstall: false}},
		{"Garbage input passed on cmdLine", fields{cmdLineInput: "GARBAGE", doesUninstall: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := createVpoDeployment(map[string]string{"app.kubernetes.io/version": "1.4.0"})
			vpo := createVpoPod()
			namespace := createNamespace()
			validatingWebhookConfig := createWebhook()
			clusterRoleBinding := createClusterRoleBinding()
			mcClusterRole := createMCClusterRole()
			registrarClusterRole := createRegistrarClusterRoleForRancher()
			vz := createVz()
			c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(
				deployment, vpo, vz, namespace, validatingWebhookConfig, clusterRoleBinding, mcClusterRole, registrarClusterRole).Build()

			if tt.fields.cmdLineInput != "" {
				content := []byte(tt.fields.cmdLineInput)
				tempfile, err := os.CreateTemp("", "test-input.txt")
				if err != nil {
					assert.Error(t, err)
				}
				// clean up tempfile
				defer os.Remove(tempfile.Name())
				if _, err := tempfile.Write(content); err != nil {
					assert.Error(t, err)
				}
				if _, err := tempfile.Seek(0, 0); err != nil {
					assert.Error(t, err)
				}
				oldStdin := os.Stdin
				// Restore original Stdin
				defer func() { os.Stdin = oldStdin }()
				os.Stdin = tempfile
			}

			// Send stdout stderr to a byte bufferF
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)
			rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
			rc.SetClient(c)
			cmd := NewCmdUninstall(rc)

			if tt.fields.doesUninstall {
				if strings.Contains(tt.name, "skip-confirmation") {
					// Suppressing uninstall prompt
					cmd.PersistentFlags().Set(ConfirmUninstallFlag, "true")
				}
				err := cmd.Execute()
				assert.NoError(t, err)
				ensureResourcesDeleted(t, c)
			} else if !tt.fields.doesUninstall {
				err := cmd.Execute()
				assert.NoError(t, err)
				ensureResourcesNotDeleted(t, c)
			}
		})
	}
}

func createNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: vzconstants.VerrazzanoInstallNamespace,
		},
	}
}

func createVz() *v1beta1.Verrazzano {
	return &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
}

func createVpoDeployment(labels map[string]string) *appsv1.Deployment {
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

func createWebhook() *adminv1.ValidatingWebhookConfiguration {
	return &adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperatorWebhook,
		},
	}
}

func createClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoPlatformOperator,
		},
	}
}

func createMCClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoManagedCluster,
		},
	}
}

func createRegistrarClusterRoleForRancher() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: vzconstants.VerrazzanoClusterRancherName,
		},
	}
}

func ensureResourcesDeleted(t *testing.T, client ctrlclient.Client) {
	// Expect the Verrazzano resource to be deleted
	v := vzapi.Verrazzano{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &v)
	assert.True(t, errors.IsNotFound(err))

	// Expect the install namespace to be deleted
	ns := corev1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoInstallNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Validating Webhook Configuration to be deleted
	vwc := adminv1.ValidatingWebhookConfiguration{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperatorWebhook}, &vwc)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role Binding to be deleted
	crb := rbacv1.ClusterRoleBinding{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &crb)
	assert.True(t, errors.IsNotFound(err))

	// Expect the managed cluster Cluster Role to be deleted
	cr := rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoManagedCluster}, &cr)
	assert.True(t, errors.IsNotFound(err))

	// Expect the cluster Registrar Cluster Role to be deleted
	cr = rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoClusterRancherName}, &cr)
	assert.True(t, errors.IsNotFound(err))
}

func ensureResourcesNotDeleted(t *testing.T, client ctrlclient.Client) {
	// Expect the install namespace not to be deleted
	ns := corev1.Namespace{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoInstallNamespace}, &ns)
	assert.NoError(t, err)

	// Expect the Validating Webhook Configuration not to be deleted
	vwc := adminv1.ValidatingWebhookConfiguration{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperatorWebhook}, &vwc)
	assert.NoError(t, err)

	// Expect the Cluster Role Binding not to be deleted
	crb := rbacv1.ClusterRoleBinding{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &crb)
	assert.NoError(t, err)

	// Expect the managed cluster Cluster Role not to be deleted
	cr := rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoManagedCluster}, &cr)
	assert.NoError(t, err)

	// Expect the cluster Registrar Cluster Role not to be deleted
	cr = rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoClusterRancherName}, &cr)
	assert.NoError(t, err)
}
