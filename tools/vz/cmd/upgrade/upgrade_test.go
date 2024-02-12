// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	dynfake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testKubeConfig     = "kubeconfig"
	testK8sContext     = "testcontext"
	testImageRegistry  = "testreg.io"
	testImagePrefix    = "testrepo"
	testVZMajorRelease = "v1.5.0"
	testVZPatchRelease = "v1.5.2"
)

// TestUpgradeCmdDefaultNoWait
// GIVEN a CLI upgrade command with all defaults and --wait==false
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful
func TestUpgradeCmdDefaultNoWait(t *testing.T) {
	vz := testhelpers.CreateVerrazzanoObjectWithVersion()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Equal(t, "", string(errBytes))

	// Verify the vz resource is as expected
	vzResource := v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vzResource)
	assert.NoError(t, err)
}

// TestUpgradeCmdDefaultTimeoutBugReport
// GIVEN a CLI upgrade command with all defaults and --timeout=2ms
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command times out and a bug report is generated
func TestUpgradeCmdDefaultTimeoutBugReport(t *testing.T) {
	vz := testhelpers.CreateVerrazzanoObjectWithVersion()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.SetDynamicClient(dynfake.NewSimpleDynamicClient(helpers.GetScheme()))
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	defer os.RemoveAll(tempKubeConfigPath.Name())

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	err = cmd.Execute()
	assert.Error(t, err)
	buf, err := os.ReadFile(rc.Out.Name())
	assert.Nil(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for upgrade to complete\n", string(errBytes))
	assert.Contains(t, string(buf), "Upgrading Verrazzano to version v1.4.0")
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal("cannot find bug report file in current directory")
	}
}

// TestUpgradeCmdDefaultTimeoutNoBugReport
// GIVEN a CLI upgrade command with all --timeout=2ms and auto-bug-report=false
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command times out and a bug report is not generated
func TestUpgradeCmdDefaultTimeoutNoBugReport(t *testing.T) {
	vz := testhelpers.CreateVerrazzanoObjectWithVersion()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "2ms")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	cmd.PersistentFlags().Set(constants.AutoBugReportFlag, "false")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	defer os.RemoveAll(tempKubeConfigPath.Name())

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	err = cmd.Execute()
	assert.Error(t, err)
	buf, err := os.ReadFile(rc.Out.Name())
	assert.Nil(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Equal(t, "Error: Timeout 2ms exceeded waiting for upgrade to complete\n", string(errBytes))
	assert.Contains(t, string(buf), "Upgrading Verrazzano to version v1.4.0")
	// Bug report must not exist
	if helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal("found bug report file in current directory")
	}
}

// TestUpgradeCmdDefaultNoVPO
// GIVEN a CLI upgrade command with all defaults and no VPO found
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command fails
func TestUpgradeCmdDefaultNoVPO(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateVerrazzanoObjectWithVersion()).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.SetDynamicClient(dynfake.NewSimpleDynamicClient(helpers.GetScheme()))
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")

	// Run upgrade command
	cmd.PersistentFlags().Set(constants.VPOTimeoutFlag, "1s")
	err = cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Contains(t, string(errBytes), "Error: Waiting for verrazzano-platform-operator pod in namespace verrazzano-install")
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal("found bug report file in current directory")
	}
}

// TestUpgradeCmdDefaultMultipleVPO
// GIVEN a CLI upgrade command with all defaults and multiple VPOs found
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command fails
func TestUpgradeCmdDefaultMultipleVPO(t *testing.T) {
	vz := testhelpers.CreateVerrazzanoObjectWithVersion()
	vpo2 := testhelpers.CreateVPOPod(constants.VerrazzanoPlatformOperator + "-2")
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz, vpo2)...).Build()
	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.SetDynamicClient(dynfake.NewSimpleDynamicClient(helpers.GetScheme()))
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	cmd.PersistentFlags().Set(constants.VPOTimeoutFlag, "1s")
	err = cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Contains(t, string(errBytes), "Error: Waiting for verrazzano-platform-operator, more than one verrazzano-platform-operator pod was found in namespace verrazzano-install")
	if !helpers.CheckAndRemoveBugReportExistsInDir("") {
		t.Fatal("found bug report file in current directory")
	}
}

// TestUpgradeCmdJsonLogFormat
// GIVEN a CLI upgrade command with defaults and --log-format=json and --wait==false
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful
func TestUpgradeCmdJsonLogFormat(t *testing.T) {
	vz := testhelpers.CreateVerrazzanoObjectWithVersion()
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.LogFormatFlag, "json")
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.Nil(t, err)
	assert.Equal(t, "", string(errBytes))
}

// TestUpgradeCmdOperatorFile
// GIVEN a CLI upgrade command with defaults and --wait=false and --operator-file specified
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful
func TestUpgradeCmdOperatorFile(t *testing.T) {
	tests := []struct {
		testName          string
		manifestsFlagName string
	}{
		{"Use manifests flag", constants.ManifestsFlag},
		{"Use deprecated operator-file flag", constants.OperatorFileFlag},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			vz := testhelpers.CreateVerrazzanoObjectWithVersion().(*v1beta1.Verrazzano)
			c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

			rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
			assert.Nil(t, err)
			defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
			rc.SetClient(c)
			cmd := NewCmdUpgrade(rc)
			assert.NotNil(t, cmd)
			cmd.PersistentFlags().Set(tt.manifestsFlagName, "../../test/testdata/operator-file-fake.yaml")
			cmd.PersistentFlags().Set(constants.WaitFlag, "false")
			cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
			defer cmdHelpers.SetDefaultDeleteFunc()

			cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
			defer cmdHelpers.SetDefaultVPOIsReadyFunc()

			// Run upgrade command
			err = cmd.Execute()
			assert.NoError(t, err)
			errBytes, err := os.ReadFile(rc.ErrOut.Name())
			assert.NoError(t, err)
			buf, err := os.ReadFile(rc.Out.Name())
			assert.NoError(t, err)
			assert.Equal(t, "", string(errBytes))
			assert.Contains(t, string(buf), "Applying the file ../../test/testdata/operator-file-fake.yaml")
			assert.Contains(t, string(buf), "namespace/verrazzano-install created")
			assert.Contains(t, string(buf), "serviceaccount/verrazzano-platform-operator created")
			assert.Contains(t, string(buf), "service/verrazzano-platform-operator created")

			// Verify the objects in the manifests got added
			sa := corev1.ServiceAccount{}
			err = c.Get(context.TODO(), types.NamespacedName{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: constants.VerrazzanoPlatformOperator}, &sa)
			assert.NoError(t, err)
			expectedLastAppliedConfigAnnotation := "{\"apiVersion\":\"v1\",\"kind\":\"ServiceAccount\",\"metadata\":{\"annotations\":{},\"name\":\"verrazzano-platform-operator\",\"namespace\":\"verrazzano-install\"}}\n"
			testhelpers.VerifyLastAppliedConfigAnnotation(t, sa.ObjectMeta, expectedLastAppliedConfigAnnotation)

			ns := corev1.Namespace{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-install"}, &ns)
			assert.NoError(t, err)
			expectedLastAppliedConfigAnnotation = "{\"apiVersion\":\"v1\",\"kind\":\"Namespace\",\"metadata\":{\"annotations\":{},\"labels\":{\"verrazzano.io/namespace\":\"verrazzano-install\"},\"name\":\"verrazzano-install\"}}\n"
			testhelpers.VerifyLastAppliedConfigAnnotation(t, ns.ObjectMeta, expectedLastAppliedConfigAnnotation)

			svc := corev1.Service{}
			err = c.Get(context.TODO(), types.NamespacedName{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: constants.VerrazzanoPlatformOperator}, &svc)
			assert.NoError(t, err)
			expectedLastAppliedConfigAnnotation = "{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"labels\":{\"app\":\"verrazzano-platform-operator\"},\"name\":\"verrazzano-platform-operator\",\"namespace\":\"verrazzano-install\"},\"spec\":{\"ports\":[{\"name\":\"http-metric\",\"port\":9100,\"protocol\":\"TCP\",\"targetPort\":9100}],\"selector\":{\"app\":\"verrazzano-platform-operator\"}}}\n"
			testhelpers.VerifyLastAppliedConfigAnnotation(t, svc.ObjectMeta, expectedLastAppliedConfigAnnotation)

			// Verify the version got updated
			err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, vz)
			assert.NoError(t, err)
			assert.Equal(t, "v1.5.2", vz.Spec.Version)
		})
	}
}

// TestUpgradeCmdNoVerrazzano
// GIVEN a CLI upgrade command with no verrazzano install resource found
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command fails
func TestUpgradeCmdNoVerrazzano(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects().Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.SetDynamicClient(dynfake.NewSimpleDynamicClient(helpers.GetScheme()))
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)

	// Run upgrade command
	err = cmd.Execute()
	assert.Error(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "Error: Verrazzano is not installed: Failed to find any Verrazzano resources\n", string(errBytes))
}

// TestUpgradeCmdLesserStatusVersion
// GIVEN a CLI upgrade command specifying a version less than the status version
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command fails
func TestUpgradeCmdLesserStatusVersion(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateVerrazzanoObjectWithVersion()).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.SetDynamicClient(dynfake.NewSimpleDynamicClient(helpers.GetScheme()))
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.3.3")

	// Run upgrade command
	err = cmd.Execute()
	assert.Error(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "Error: Upgrade to a lesser version of Verrazzano is not allowed. Upgrade version specified was v1.3.3 and current Verrazzano version is v1.3.4\n", string(errBytes))
}

// TestUpgradeCmdEqualStatusVersion
// GIVEN a CLI upgrade command specifying a version equal to the status version and the spec version is empty
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful with an informational message
func TestUpgradeCmdEqualStatusVersion(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateVerrazzanoObjectWithVersion()).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.3.4")

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	buf, err := os.ReadFile(rc.Out.Name())
	assert.NoError(t, err)
	assert.Equal(t, "Verrazzano is already at the specified upgrade version of v1.3.4\n", string(buf))
}

// TestUpgradeCmdLesserSpecVersion
// GIVEN a CLI upgrade command specifying a version less than the spec version
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command fails
func TestUpgradeCmdLesserSpecVersion(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Spec: v1beta1.VerrazzanoSpec{
			Version: "v1.3.4",
		},
		Status: v1beta1.VerrazzanoStatus{
			Version: "v1.3.3",
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vz).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.3.3")

	// Run upgrade command
	err = cmd.Execute()
	assert.Error(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "Error: Upgrade to a lesser version of Verrazzano is not allowed. Upgrade version specified was v1.3.3 and the upgrade in progress is v1.3.4\n", string(errBytes))
}

// TestUpgradeCmdInProgress
// GIVEN a CLI upgrade command an upgrade was in progress
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful
func TestUpgradeCmdInProgress(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
		Spec: v1beta1.VerrazzanoSpec{
			Version: "v1.3.4",
		},
		Status: v1beta1.VerrazzanoStatus{
			Version: "v1.3.3",
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.3.4")

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))
}

// TestUpgradeFromPrivateRegistry tests upgrading from a private registry.
//
// GIVEN Verrazzano is installed from a private registry
//
//	WHEN I call cmd.Execute for upgrade with the same private registry settings
//	THEN the CLI upgrade command is successful and the VPO and VPO webhook deployments have the expected private registry configuration
func TestUpgradeFromPrivateRegistry(t *testing.T) {
	// First install using a private registry

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := install.NewCmdInstall(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZMajorRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, testImageRegistry)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, testImagePrefix)
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	install.SetValidateCRFunc(install.FakeValidateCRFunc)
	defer install.SetDefaultValidateCRFunc()

	// Run install command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))

	// Need to update the VZ status version otherwise upgrade fails
	vz := &v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, vz)
	assert.NoError(t, err)

	vz.Status.Version = testVZMajorRelease
	err = c.Status().Update(context.TODO(), vz)
	assert.NoError(t, err)

	// Now do the upgrade using the same private registry settings
	cmd = NewCmdUpgrade(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZPatchRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, testImageRegistry)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, testImagePrefix)

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err = os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))

	// Verify that the VPO deployment has the expected environment variables to enable pulling images from a private registry
	deployment, err := cmdHelpers.GetExistingVPODeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)
	testhelpers.AssertPrivateRegistryEnvVars(t, c, deployment, testImageRegistry, testImagePrefix)

	// Verify that the VPO image has been updated
	testhelpers.AssertPrivateRegistryImage(t, c, deployment, testImageRegistry, testImagePrefix)

	// Verify that the VPO webhook image has been updated
	deployment, err = cmdHelpers.GetExistingVPOWebhookDeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	testhelpers.AssertPrivateRegistryImage(t, c, deployment, testImageRegistry, testImagePrefix)
}

// TestUpgradeFromDifferentPrivateRegistry tests upgrading from a different private registry
func TestUpgradeFromDifferentPrivateRegistry(t *testing.T) {
	// First install using a private registry
	const proceedQuestionText = "Proceed to upgrade with new settings? [y/N]"
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)

	content := []byte("y")
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

	cmd := install.NewCmdInstall(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZMajorRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, testImageRegistry)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, testImagePrefix)
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	install.SetValidateCRFunc(install.FakeValidateCRFunc)
	defer install.SetDefaultValidateCRFunc()

	// Run install command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))

	// Need to update the VZ status version otherwise upgrade fails
	vz := &v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, vz)
	assert.NoError(t, err)

	vz.Status.Version = testVZMajorRelease
	err = c.Status().Update(context.TODO(), vz)
	assert.NoError(t, err)
	testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)

	// GIVEN Verrazzano is installed from a private registry
	//
	//	WHEN I call cmd.Execute for upgrade with different private registry settings and answer "n" when asked to proceed
	//	THEN the upgrade is cancelled
	const imageRegistryForUpgrade = "newreg.io"
	const imagePrefixForUpgrade = "newrepo"

	// Create a buffer for Stdin that simulates the user typing an "n" in response to the question on whether the CLI
	// should continue with the upgrade because the registry settings are different from the settings used during install
	inputFile, err := os.CreateTemp("", "tmpstdin")
	assert.Nil(t, err)
	inputFile.WriteString("n\n")
	defer os.Remove(inputFile.Name())
	rc, err = testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.IOStreams.In = inputFile
	rc.SetClient(c)

	cmd = NewCmdUpgrade(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZPatchRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, imageRegistryForUpgrade)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, imagePrefixForUpgrade)

	// Run upgrade command - expect that the CLI asks us if we want to continue with the existing registry settings
	// and we reply with "n"
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err = os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	outBytes, err := os.ReadFile(rc.Out.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(outBytes), proceedQuestionText)
	assert.Contains(t, string(outBytes), "Upgrade canceled")
	assert.Equal(t, "", string(errBytes))

	// Verify that the VPO deployment has the expected environment variables to enable pulling images from a private registry
	// and that they are the settings from the install, not the upgrade
	deployment, err := cmdHelpers.GetExistingVPODeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)
	testhelpers.AssertPrivateRegistryEnvVars(t, c, deployment, testImageRegistry, testImagePrefix)

	// Verify that the VPO image is using the install private registry settings
	testhelpers.AssertPrivateRegistryImage(t, c, deployment, testImageRegistry, testImagePrefix)

	// Verify that the VPO webhook image is using the install private registry settings
	deployment, err = cmdHelpers.GetExistingVPOWebhookDeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	testhelpers.AssertPrivateRegistryImage(t, c, deployment, testImageRegistry, testImagePrefix)
	testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)

	// GIVEN Verrazzano is installed from a private registry
	//
	//	WHEN I call cmd.Execute for upgrade with different private registry settings and answer "y" when asked to proceed
	//	THEN the upgrade succeeds and the new registry settings are configured on the VPO
	inputFile, err = os.CreateTemp("", "tmpstdin")
	assert.Nil(t, err)
	inputFile.WriteString("y\n")
	defer os.Remove(inputFile.Name())
	rc, err = testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	rc.In = inputFile

	cmd = NewCmdUpgrade(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZPatchRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, imageRegistryForUpgrade)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, imagePrefixForUpgrade)

	// Run upgrade command - expect that the CLI asks us if we want to continue with the existing registry settings
	// and we reply with "y"
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.NoError(t, err)
	errBytes, err = os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	outBytes, err = os.ReadFile(rc.Out.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(outBytes), proceedQuestionText)
	assert.Contains(t, string(outBytes), "Upgrading Verrazzano")
	assert.Equal(t, "", string(errBytes))

	// Verify that the VPO deployment has the expected environment variables to enable pulling images from a private registry
	// and that they are the new settings from the upgrade
	deployment, err = cmdHelpers.GetExistingVPODeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)
	testhelpers.AssertPrivateRegistryEnvVars(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)

	// Verify that the VPO image is using the upgrade private registry settings
	testhelpers.AssertPrivateRegistryImage(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)

	// Verify that the VPO webhook image is using the upgrade private registry settings
	deployment, err = cmdHelpers.GetExistingVPOWebhookDeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	testhelpers.AssertPrivateRegistryImage(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)
}

// TestUpgradeFromPrivateRegistryWithForce tests upgrading from a private registry to a different private registry using the skip confirmation flag
//
// GIVEN Verrazzano is installed from a private registry
//
//	WHEN I call cmd.Execute for upgrade with different private registry settings and I set the skip confirmation flag
//	THEN the CLI upgrade command is successful and the VPO and VPO webhook deployments have the expected private registry configuration
func TestUpgradeFromPrivateRegistryWithSkipConfirmation(t *testing.T) {
	// First install using a private registry
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)
	cmd := install.NewCmdInstall(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZMajorRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, testImageRegistry)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, testImagePrefix)
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()

	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	install.SetValidateCRFunc(install.FakeValidateCRFunc)
	defer install.SetDefaultValidateCRFunc()

	// Run install command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))

	// Need to update the VZ status version otherwise upgrade fails
	vz := &v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, vz)
	assert.NoError(t, err)

	vz.Status.Version = testVZMajorRelease
	err = c.Status().Update(context.TODO(), vz)
	assert.NoError(t, err)
	testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)

	// Now do the upgrade using different private registry settings
	const imageRegistryForUpgrade = "newreg.io"
	const imagePrefixForUpgrade = "newrepo"

	rc, err = testhelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rc.SetClient(c)

	cmd = NewCmdUpgrade(rc)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")
	cmd.PersistentFlags().Set(constants.VersionFlag, testVZPatchRelease)
	cmd.PersistentFlags().Set(constants.ImageRegistryFlag, imageRegistryForUpgrade)
	cmd.PersistentFlags().Set(constants.ImagePrefixFlag, imagePrefixForUpgrade)

	// Set the "skip confirmation" flag so we do not get asked to continue with the upgrade
	cmd.PersistentFlags().Set(constants.SkipConfirmationFlag, "true")

	// Run upgrade command
	err = cmd.Execute()
	assert.NoError(t, err)
	errBytes, err = os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	outBuf, err := os.ReadFile(rc.Out.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(outBuf), "Upgrading Verrazzano")
	assert.Equal(t, "", string(errBytes))

	// Verify that the VPO deployment environment variables for private registry have been updated
	deployment, err := cmdHelpers.GetExistingVPODeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)
	testhelpers.AssertPrivateRegistryEnvVars(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)

	// Verify that the VPO image has been updated
	testhelpers.AssertPrivateRegistryImage(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)

	// Verify that the VPO webhook image has been updated
	deployment, err = cmdHelpers.GetExistingVPOWebhookDeployment(c)
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	testhelpers.AssertPrivateRegistryImage(t, c, deployment, imageRegistryForUpgrade, imagePrefixForUpgrade)
}

// TestUpgradeCmdWithSetFlagsNoWait
// GIVEN a CLI upgrade command with all defaults and --wait==false
//
//	WHEN I call cmd.Execute for upgrade
//	THEN the CLI upgrade command is successful
func TestUpgradeCmdWithSetFlagsNoWait(t *testing.T) {
	tests := []struct {
		name           string
		isValidSetArgs bool
		setArgs        []string
	}{
		{name: "Upgrade with valid --set args", isValidSetArgs: true, setArgs: []string{"spec.components.jaegerOperator.enabled=true"}},
		{name: "Upgrade with invalid --set args", isValidSetArgs: false, setArgs: []string{"s"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vz := testhelpers.CreateVerrazzanoObjectWithVersion()
			c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(append(testhelpers.CreateTestVPOObjects(), vz)...).Build()

			// Send stdout stderr to a byte buffer
			rc, err := testhelpers.NewFakeRootCmdContextWithFiles()
			assert.Nil(t, err)
			defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
			rc.SetClient(c)
			cmd := NewCmdUpgrade(rc)
			assert.NotNil(t, cmd)
			cmd.PersistentFlags().Set(constants.WaitFlag, "false")
			cmd.PersistentFlags().Set(constants.VersionFlag, "v1.4.0")
			for _, arg := range tt.setArgs {
				cmd.PersistentFlags().Set(constants.SetFlag, arg)
			}
			cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
			defer cmdHelpers.SetDefaultDeleteFunc()

			cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
			defer cmdHelpers.SetDefaultVPOIsReadyFunc()

			// Run upgrade command
			err = cmd.Execute()

			if tt.isValidSetArgs {
				// Verify the vz resource is as expected
				vzResource := v1beta1.Verrazzano{}
				err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &vzResource)
				assert.NoError(t, err)
				isJaegerOperatorEnabled := vzResource.Spec.Components.JaegerOperator.Enabled
				assert.NotNil(t, isJaegerOperatorEnabled)
				assert.Equal(t, true, *isJaegerOperatorEnabled)
			} else {
				assert.Error(t, err)
			}

		})
	}

}
