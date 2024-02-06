// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/stretchr/testify/assert"
	appv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	appclusterv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	appoamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	clusterv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// TestCaptureK8SResources
//
//	WHEN I call functions to capture k8s resource
//	THEN expect it to not throw any error
func TestCaptureK8SResources(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset()
	scheme := k8scheme.Scheme
	AddCapiToScheme(scheme)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme)
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp("", "testfile")
	defer cleanupFile(t, tempFile)
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = CaptureK8SResources(k8sClient, dynamicClient, constants.VerrazzanoInstall, captureDir, rc)
	assert.NoError(t, err)
	isError = false
}

// TestCaptureMultiClusterResources tests the functionality to capture the multi cluster related resources
//
//	WHEN I call functions to capture Verrazzano multi cluster resources
//	THEN expect it to not throw any error
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
	assert.NoError(t, CaptureMultiClusterOAMResources(dynamicClient, []string{constants.VerrazzanoInstall}, captureDir, rc))
}

// TestCaptureOAMResources tests the functionality to capture the OAM resources in the cluster
//
//	WHEN I call functions to capture Verrazzano OAM resources
//	THEN expect it to not throw any error
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

// TestCapturePodLog tests the functionality to capture the logs of a given pod.
func TestCapturePodLog(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset()
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = CapturePodLog(k8sClient, corev1.Pod{}, constants.VerrazzanoInstall, captureDir, rc, 0)
	assert.NoError(t, err)

	//  GIVENT and empty k8s cluster,
	//	WHEN I call functions to capture VPO pod logs,
	//	THEN expect it to not throw any error.
	err = CapturePodLog(k8sClient, corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}}, constants.VerrazzanoInstall, captureDir, rc, 0)
	assert.NoError(t, err)

	//  GIVENT a k8s cluster with a VPO pod,
	//	WHEN I call functions to capture VPO pod logs,
	//	THEN expect it to not throw any error.
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
	}}, constants.VerrazzanoInstall, captureDir, rc, 300)
	assert.NoError(t, err)
}

// TestGetPodList tests the functionality to return the list of pods with the given label
func TestGetPodList(t *testing.T) {
	//  GIVEN a k8s cluster with no VPO pods,
	//	WHEN I call functions to get the list of pods in the k8s cluster,
	//	THEN expect it to be an empty list.
	pods, err := GetPodList(fake.NewClientBuilder().Build(), "app", constants.VerrazzanoPlatformOperator, constants.VerrazzanoInstall)
	assert.NoError(t, err)
	assert.Empty(t, pods)

	//  GIVEN a k8s cluster with a VPO pod,
	//	WHEN I call functions to get the list of pods in the k8s cluster,
	//	THEN expect it to be an empty list.
	pods, err = GetPodList(fake.NewClientBuilder().WithObjects(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VerrazzanoPlatformOperator,
			Namespace: constants.VerrazzanoInstall,
			Labels:    map[string]string{"app": constants.VerrazzanoPlatformOperator},
		},
	}).Build(), "app", constants.VerrazzanoPlatformOperator, constants.VerrazzanoInstall)
	assert.NoError(t, err)
	assert.NotEmpty(t, pods)
}

// TestCaptureVZResource tests the functionality to capture the Verrazzano resource.
func TestCaptureVZResource(t *testing.T) {
	captureDir, err := os.MkdirTemp("", "testcapture")
	defer cleanupTempDir(t, captureDir)
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	//  GIVEN a k8s cluster with a user provided Verrazzano CR,
	//	WHEN I call functions to capture the Verrazzano CR,
	//	THEN expect the file to contain the JSON output of the Verrazzano CR.
	vz := &v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "myverrazzano",
		},
		Spec: v1beta1.VerrazzanoSpec{
			Profile: v1beta1.Dev,
		},
	}
	tempFile, err := os.CreateTemp("", "testfile")
	defer cleanupFile(t, tempFile)
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	SetVerboseOutput(true)
	SetIsLiveCluster()
	err = CaptureVZResource(captureDir, vz)
	assert.NoError(t, err)
	assert.NotNil(t, GetMultiWriterOut())
	assert.NotNil(t, GetMultiWriterErr())
	assert.True(t, GetIsLiveCluster())
}

// TestDoesNamespaceExist tests the functionality to check if a given namespace exists.
func TestDoesNamespaceExist(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	tempFile, _ := os.CreateTemp("", "testfile")
	defer cleanupFile(t, tempFile)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	SetVerboseOutput(true)

	//  GIVEN a k8s cluster with no namespaces,
	//	WHEN I call functions to check if a namespace with empty string exists,
	//	THEN expect it to return false and no error.
	exists, err := DoesNamespaceExist(k8sfake.NewSimpleClientset(), "", rc)
	assert.NoError(t, err)
	assert.False(t, exists)

	//  GIVEN a k8s cluster with no namespaces,
	//	WHEN I call functions to check if a namespace verrazzano-install exists,
	//	THEN expect it to return false and an error.
	exists, err = DoesNamespaceExist(k8sfake.NewSimpleClientset(), constants.VerrazzanoInstall, rc)
	assert.Error(t, err)
	assert.False(t, exists)

	//  GIVEN a k8s cluster with the required verrazzano-install namespace,
	//	WHEN I call functions to check if a namespace verrazzano-install exists,
	//	THEN expect it to return true and no error.
	exists, err = DoesNamespaceExist(k8sfake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: constants.VerrazzanoInstall,
	}}), constants.VerrazzanoInstall, rc)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestGetVZManagedNamespaces tests the functionality to return all namespaces managed by verrazzano
func TestGetVZManagedNamespaces(t *testing.T) {
	namespaces := GetVZManagedNamespaces(k8sfake.NewSimpleClientset())
	assert.Empty(t, namespaces)

	//  GIVEN a k8s cluster with the required verrazzano-install namespace with label verrazzano-managed=true,
	//	WHEN I call functions to list the namespaces that are managed by Verrazzano,
	//	THEN expect it to return a single namespace verrazzano-install
	namespaces = GetVZManagedNamespaces(k8sfake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   constants.VerrazzanoInstall,
		Labels: map[string]string{"verrazzano-managed": "true"},
	}}))
	assert.NotEmpty(t, namespaces)
	assert.Equal(t, 1, len(namespaces))
	assert.Equal(t, constants.VerrazzanoInstall, namespaces[0])
}

// TestIsErrorReported tests the functionality to see if an error had been reported when capturing the k8s resources.
func TestIsErrorReported(t *testing.T) {
	assert.False(t, IsErrorReported())
	LogError("dummy error msg")
	assert.True(t, IsErrorReported())
}

// TestCreateFile tests the functionality to create a file containing the Verrazzano Resource
func TestCreateFile(t *testing.T) {
	//  GIVEN a k8s cluster with a VPO pod,
	//	WHEN I call functions to create a JSON file for the pod,
	//	THEN expect it to write to the provided resource file, the JSON contents of the pod and no error should be returned.
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

// cleanupTempDir cleans up the given temp directory after the test run
func cleanupTempDir(t *testing.T, dirName string) {
	if err := os.RemoveAll(dirName); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
}

// cleanupTempDir cleans up the given temp file after the test run
func cleanupFile(t *testing.T, file *os.File) {
	if err := file.Close(); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
}

// TestGetPodListAll tests the functionality to return the list of all pods
func TestGetPodListAll(t *testing.T) {
	nsName := "test"
	podLength := 5
	var podList []client.Object
	for i := 0; i < podLength; i++ {
		podList = append(podList, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsName + fmt.Sprint(i),
				Namespace: nsName,
				Labels:    map[string]string{"name": "myapp"},
			},
		})
	}
	//  GIVEN a k8s cluster with no pods,
	//	WHEN I call functions to get the list of pods in the k8s cluster,
	//	THEN expect it to be an empty list.
	pods, err := GetPodListAll(fake.NewClientBuilder().Build(), nsName)
	assert.NoError(t, err)
	assert.Empty(t, pods)

	//  GIVEN a k8s cluster with 5 pods,
	//	WHEN I call functions to get the list of pods in the k8s cluster without label,
	//	THEN expect it to be list all pods.
	pods, err = GetPodListAll(fake.NewClientBuilder().WithObjects(podList...).Build(), nsName)
	assert.NoError(t, err)
	assert.Equal(t, podLength, len(pods))
}

// TestCreateNamespaceFile tests that a namespace file titled namespace.json can be successfully written
// GIVEN a k8s cluster,
// WHEN I call functions to get the namespace resource for that namespace
// THEN expect it to write to the provided resource file with the correct information and no error should be returned.
func TestCreateNamespaceFile(t *testing.T) {
	listOfFinalizers := []corev1.FinalizerName{"test-finalizer-name"}
	sampleNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: listOfFinalizers,
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceTerminating,
		},
	}
	client := k8sfake.NewSimpleClientset(&sampleNamespace)
	captureDir, err := os.MkdirTemp("", "testcapturefornamespaces")
	assert.NoError(t, err)
	t.Log(captureDir)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = captureNamespaces(client, "test-namespace", captureDir, rc)
	assert.NoError(t, err)
	namespaceLocation := filepath.Join(captureDir, "test-namespace", constants.NamespaceJSON)
	namespaceObjectToUnmarshalInto := &corev1.Namespace{}
	err = unmarshallFile(namespaceLocation, namespaceObjectToUnmarshalInto)
	assert.NoError(t, err)
	assert.True(t, namespaceObjectToUnmarshalInto.Status.Phase == corev1.NamespaceTerminating)
}

// TestCreateMetadataFile tests that a metadata file titled metadata.json can be successfully written
// GIVEN a k8s cluster,
// WHEN I call functions to record the metadata for the capture
// THEN expect it to write to the correct resource file with the correct information and no error should be returned.
func TestCreateMetadataFile(t *testing.T) {
	captureDir, err := os.MkdirTemp("", "testcaptureformetadata")
	assert.NoError(t, err)
	t.Log(captureDir)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	SetMultiWriterErr(errBuf, tempFile)
	err = CaptureMetadata(captureDir)
	assert.NoError(t, err)
	metadataLocation := filepath.Join(captureDir, constants.MetadataJSON)
	metadataObjectToUnmarshalInto := &Metadata{}
	err = unmarshallFile(metadataLocation, metadataObjectToUnmarshalInto)
	assert.NoError(t, err)
	timeObject, err := time.Parse(time.RFC3339, metadataObjectToUnmarshalInto.Time)
	assert.NoError(t, err)
	assert.True(t, time.Now().UTC().Sub(timeObject).Minutes() < 60)

}

// unmarshallFile is a helper function that is used to place the contents of a file into a defined struct
func unmarshallFile(clusterPath string, object interface{}) error {
	// Parse the json into local struct
	file, err := os.Open(clusterPath)
	if os.IsNotExist(err) {
		// The file may not exist if the component is not installed.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open file %s from cluster snapshot: %s", clusterPath, err.Error())
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Failed reading Json file %s: %s", clusterPath, err.Error())
	}

	// Unmarshall file contents into a struct
	err = json.Unmarshal(fileBytes, object)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s: %s", clusterPath, err.Error())
	}

	return nil
}
