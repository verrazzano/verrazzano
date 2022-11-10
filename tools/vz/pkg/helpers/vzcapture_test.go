// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"os"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/stretchr/testify/assert"
	appv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	appclusterv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	appoamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	clusterv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateReportArchive
// GIVEN a directory containing some files
//
//	WHEN I call function CreateReportArchive with a report file
//	THEN expect it to create the report file
func TestCreateReportArchive(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "bug-report")
	defer cleanupTempDir(t, tmpDir)

	captureDir := tmpDir + string(os.PathSeparator) + "test-report"
	if err := os.Mkdir(captureDir, os.ModePerm); err != nil {
		assert.Error(t, err)
	}

	// Create some files inside bugReport
	_, err := os.Create(captureDir + string(os.PathSeparator) + "f1.txt")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = os.Create(captureDir + string(os.PathSeparator) + "f2.txt")
	if err != nil {
		assert.Error(t, err)
	}

	_, err = os.Create(captureDir + string(os.PathSeparator) + "f3.txt")
	if err != nil {
		assert.Error(t, err)
	}

	bugReportFile, err := os.Create(tmpDir + string(os.PathSeparator) + "bug.tar.gz")
	if err != nil {
		assert.Error(t, err)
	}
	err = CreateReportArchive(captureDir, bugReportFile)
	if err != nil {
		assert.Error(t, err)
	}

	// Check file exists
	assert.FileExists(t, bugReportFile.Name())
}

// TestRemoveDuplicates
// GIVEN a string slice containing duplicates
//
//	WHEN I call function RemoveDuplicate
//	THEN expect it to remove the duplicate elements
func TestRemoveDuplicates(t *testing.T) {
	testSlice := []string{"abc", "def", "abc"}
	result := RemoveDuplicate(testSlice)
	assert.True(t, true, len(result) == 2)
}

// TestGroupVersionResource
//
//	WHEN I call functions to get the config schemes
//	THEN expect it to return the expected resource
func TestGroupVersionResource(t *testing.T) {
	assert.True(t, true, GetAppConfigScheme().Resource == constants.OAMAppConfigurations)
	assert.True(t, true, GetComponentConfigScheme().Resource == constants.OAMComponents)
	assert.True(t, true, GetMetricsTraitConfigScheme().Resource == constants.OAMMetricsTraits)
	assert.True(t, true, GetIngressTraitConfigScheme().Resource == constants.OAMIngressTraits)
	assert.True(t, true, GetMCComponentScheme().Resource == constants.OAMMCCompConfigurations)
	assert.True(t, true, GetMCAppConfigScheme().Resource == constants.OAMMCAppConfigurations)
	assert.True(t, true, GetVzProjectsConfigScheme().Resource == constants.OAMProjects)
	assert.True(t, true, GetManagedClusterConfigScheme().Resource == constants.OAMManagedClusters)
}

func TestCaptureK8SResources(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset()
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = CaptureK8SResources(k8sClient, constants.VerrazzanoInstall, captureDir, rc)
	assert.NoError(t, err)
}
func TestCaptureMultiClusterResources(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = v1beta1.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)
	_ = appclusterv1alpha1.AddToScheme(scheme)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme)
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	assert.NoError(t, CaptureMultiClusterResources(dynamicClient, []string{constants.VerrazzanoInstall}, captureDir, rc))
}

func TestCaptureOAMResources(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = v1beta1.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)
	_ = appclusterv1alpha1.AddToScheme(scheme)
	_ = appv1alpha1.AddToScheme(scheme)
	_ = appoamv1alpha1.AddToScheme(scheme)
	_ = core.AddToScheme(scheme)

	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme)
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	assert.NoError(t, CaptureOAMResources(dynamicClient, []string{constants.VerrazzanoInstall}, captureDir, rc))
}

func TestCapturePodLog(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset()
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = CapturePodLog(k8sClient, corev1.Pod{}, constants.VerrazzanoInstall, captureDir, rc)
	assert.NoError(t, err)

	err = CapturePodLog(k8sClient, corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}}, constants.VerrazzanoInstall, captureDir, rc)
	assert.NoError(t, err)

	k8sClient = k8sfake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}, Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "testcontainer",
				Image: "dummimage:notag",
			},
		},
	}})
	err = CapturePodLog(k8sClient, corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}, Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "testcontainer",
				Image: "dummimage:notag",
			},
		},
	}}, constants.VerrazzanoInstall, captureDir, rc)
	assert.NoError(t, err)
}

func TestGetPodList(t *testing.T) {
	pods, err := GetPodList(fake.NewClientBuilder().Build(), "app", constants.VerrazzanoPlatformOperator, constants.VerrazzanoInstall)
	assert.NoError(t, err)
	assert.Empty(t, pods)
}
func TestCaptureVZResource(t *testing.T) {
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	vzList := v1beta1.VerrazzanoList{
		Items: []v1beta1.Verrazzano{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "myverrazzano",
				},
				Spec: v1beta1.VerrazzanoSpec{
					Profile: v1beta1.Dev,
				},
			},
		},
	}
	tempFile, err := os.CreateTemp("", "testfile")
	defer cleanupFile(t, tempFile)
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	SetVerboseOutput(true)
	SetIsLiveCluster()
	err = CaptureVZResource(captureDir, vzList, rc)
	assert.NoError(t, err)
	assert.NotNil(t, GetMultiWriterOut())
	assert.NotNil(t, GetMultiWriterErr())
	assert.True(t, GetIsLiveCluster())
}

func TestDoesNamespaceExist(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	tempFile, _ := os.CreateTemp("", "testfile")
	defer cleanupFile(t, tempFile)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	SetVerboseOutput(true)

	exists, err := DoesNamespaceExist(k8sfake.NewSimpleClientset(), "", rc)
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = DoesNamespaceExist(k8sfake.NewSimpleClientset(), constants.VerrazzanoInstall, rc)
	assert.Error(t, err)
	assert.False(t, exists)

	exists, err = DoesNamespaceExist(k8sfake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: constants.VerrazzanoInstall,
	}}), constants.VerrazzanoInstall, rc)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestGetVZManagedNamespaces(t *testing.T) {
	namespaces := GetVZManagedNamespaces(k8sfake.NewSimpleClientset())
	assert.Empty(t, namespaces)

	namespaces = GetVZManagedNamespaces(k8sfake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   constants.VerrazzanoInstall,
		Labels: map[string]string{"verrazzano-managed": "true"},
	}}))
	assert.NotEmpty(t, namespaces)
	assert.Equal(t, 1, len(namespaces))
	assert.Equal(t, constants.VerrazzanoInstall, namespaces[0])
}

func TestIsErrorReported(t *testing.T) {
	assert.False(t, IsErrorReported())
	LogError("dummy error msg")
	assert.True(t, IsErrorReported())
}

func TestCreateFile(t *testing.T) {
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = createFile(corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}}, constants.VerrazzanoInstall, "test-file", captureDir, rc)
	assert.NoError(t, err)
}

func cleanupTempDir(t *testing.T, dirName string) {
	if err := os.RemoveAll(dirName); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
}

func cleanupFile(t *testing.T, file *os.File) {
	if err := file.Close(); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
}
