// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/stretchr/testify/assert"
	appv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	appclusterv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	appoamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	clusterv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
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

const dummyIP1 = "0.0.0.0"
const dummyIP2 = "5.6.x.x"

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
	err = CreateReportArchive(captureDir, bugReportFile, true)
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
	schemeForClient := k8scheme.Scheme
	err := certmanagerv1.AddToScheme(schemeForClient)
	assert.NoError(t, err)
	k8sClient := k8sfake.NewSimpleClientset()
	scheme := k8scheme.Scheme
	AddCapiToScheme(scheme)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme)
	client := fake.NewClientBuilder().WithScheme(schemeForClient).Build()
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
	err = CaptureK8SResources(client, k8sClient, dynamicClient, constants.VerrazzanoInstall, captureDir, rc)
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
	err = CapturePodLog(k8sClient, corev1.Pod{}, constants.VerrazzanoInstall, captureDir, rc, 0, false)
	assert.NoError(t, err)

	//  GIVENT and empty k8s cluster,
	//	WHEN I call functions to capture VPO pod logs,
	//	THEN expect it to not throw any error.
	err = CapturePodLog(k8sClient, corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      constants.VerrazzanoPlatformOperator,
		Namespace: constants.VerrazzanoInstall,
	}}, constants.VerrazzanoInstall, captureDir, rc, 0, false)
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
	}}, constants.VerrazzanoInstall, captureDir, rc, 300, false)
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

// TestCreateInnoDBClusterFile tests that a InnoDBCluster file titled inno-db-cluster.json can be successfully written
// GIVEN a k8s cluster with a inno-db-cluster resources present in a namespace,
// WHEN I call functions to create a list of inno-db-cluster resources for the namespace
// THEN expect it to write to the provided resource file and no error should be returned
func TestCreateInnoDBClusterFile(t *testing.T) {
	schemeForClient := runtime.NewScheme()
	innoDBClusterGVK := schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	}
	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)
	innoDBCluster.SetNamespace("keycloak")
	innoDBCluster.SetName("my-sql")
	innoDBClusterStatusFields := []string{"status", "cluster", "status"}
	err := unstructured.SetNestedField(innoDBCluster.Object, "ONLINE", innoDBClusterStatusFields...)
	assert.NoError(t, err)
	metav1Time := metav1.Time{
		Time: time.Now().UTC(),
	}
	innoDBCluster.SetDeletionTimestamp(&metav1Time)
	cli := fake.NewClientBuilder().WithScheme(schemeForClient).WithObjects(&innoDBCluster).Build()
	captureDir, err := os.MkdirTemp("", "testcaptureforinnodbclusters")
	assert.Nil(t, err)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = captureInnoDBClusterResources(cli, "keycloak", captureDir, rc)
	assert.NoError(t, err)
	innoDBClusterLocation := filepath.Join(captureDir, "keycloak", constants.InnoDBClusterJSON)
	innoDBClusterListToUnmarshalInto := unstructured.UnstructuredList{}
	bytesToUnmarshall, err := returnBytesFromAFile(innoDBClusterLocation)
	assert.NoError(t, err)
	innoDBClusterListToUnmarshalInto.UnmarshalJSON(bytesToUnmarshall)
	assert.NoError(t, err)
	innoDBClusterResource := innoDBClusterListToUnmarshalInto.Items[0]
	statusOfCluster, _, err := unstructured.NestedString(innoDBClusterResource.Object, "status", "cluster", "status")
	assert.NoError(t, err)
	assert.Equal(t, statusOfCluster, "ONLINE")

}

//		TestCreateCertificateFile tests that a certificate file titled certificates.json can be successfully written
//	 	GIVEN a k8s cluster with certificates present in a namespace,
//		WHEN I call functions to create a list of certificates for the namespace,
//		THEN expect it to write to the provided resource file and no error should be returned.
func TestCreateCertificateFile(t *testing.T) {
	schemeForClient := k8scheme.Scheme
	err := certmanagerv1.AddToScheme(schemeForClient)
	assert.NoError(t, err)
	sampleCert := certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: "testcertificate", Namespace: "cattle-system"},
		Spec: certmanagerv1.CertificateSpec{
			DNSNames:    []string{"example.com", "www.example.com", "api.example.com"},
			IPAddresses: []string{dummyIP1, dummyIP2},
		},
	}
	client := fake.NewClientBuilder().WithScheme(schemeForClient).WithObjects(&sampleCert).Build()
	captureDir, err := os.MkdirTemp("", "testcaptureforcertificates")
	assert.NoError(t, err)
	t.Log(captureDir)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = captureCertificates(client, "cattle-system", captureDir, rc)
	assert.NoError(t, err)
}

// TestCreateCaCrtInfoFile tests that a caCrtInfo file titled caCrtInfo.json can be successfully written
// GIVEN a k8s cluster with secrets containing caCrtInfo present in a namespace  ,
// WHEN I call functions to create a list of caCrt for the namespace,
// THEN expect it to write to the provided resource file and no error should be returned.
func TestCreateCaCrtJsonFile(t *testing.T) {
	schemeForClient := k8scheme.Scheme
	err := certmanagerv1.AddToScheme(schemeForClient)
	assert.NoError(t, err)
	certificateListForTest := certmanagerv1.CertificateList{}
	sampleCert := certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: "test-certificate-caCrt.json", Namespace: "cattle-system"},
		Spec: certmanagerv1.CertificateSpec{
			DNSNames:    []string{"example.com", "www.example.com", "api.example.com"},
			IPAddresses: []string{dummyIP1, dummyIP2},
			SecretName:  "test-secret-name",
		},
	}
	certificateListForTest.Items = append(certificateListForTest.Items, sampleCert)
	sampleSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret-name", Namespace: "cattle-system"},
		Data: map[string][]byte{
			"ca.crt": []byte("-----BEGIN CERTIFICATE-----\nMIIDvTCCAqWgAwIBAgIUJUr+YG3UuQJh6g4MKuRpZPnTVO0wDQYJKoZIhvcNAQEL\nBQAwbTELMAkGA1UEBhMCUEExDTALBgNVBAgMBFRlc3QxDTALBgNVBAcMBFRlc3Qx\nDTALBgNVBAoMBFRlc3QxDTALBgNVBAsMBFRlc3QxDTALBgNVBAMMBFRlc3QxEzAR\nBgkqhkiG9w0BCQEWBFRlc3QwIBcNMjMwODMwMTkxMjE4WhgPMzAyMjEyMzExOTEy\nMThaMG0xCzAJBgNVBAYTAlBBMQ0wCwYDVQQIDARUZXN0MQ0wCwYDVQQHDARUZXN0\nMQ0wCwYDVQQKDARUZXN0MQ0wCwYDVQQLDARUZXN0MQ0wCwYDVQQDDARUZXN0MRMw\nEQYJKoZIhvcNAQkBFgRUZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKC\nAQEA5R5EAPbPhrfRnpGtC49OX9q4XDVP11C/nHZ13z4QMPQn3eD+S5DODjo95wVD\nbZlmOUdGhas037W/G4rsEr+fg2DF3tNV3bNtDU5NG+PjRDcmFDKup0q7Lh7Yf2FP\naNxi6wlIgmm8Yi4lQmaBSN5LZalIbTO+tk7PRa1FY2LCIKDzzY9ipc0h9nDQWXIz\nEUtjQdQuZsdcv+br2L6b891Pu/fiZgJg1Vzx8N9bBbxMl3usI/CT8qmJy4E9fh4q\n0LQMFcOXeVSR4dhGLpctXP82AH2wgz0mLmgXlYe3koX+TlOxGIG3tUKBndvII8wm\nO03wILuk63XhXg30EFjpj0qZiQIDAQABo1MwUTAdBgNVHQ4EFgQUxkWW0nvivNEy\nLAPMJYgNwpSHQ5IwHwYDVR0jBBgwFoAUxkWW0nvivNEyLAPMJYgNwpSHQ5IwDwYD\nVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAtUjYhkDzJoFNx84Y9KXJ\nVM5BRtiI7YuvrKujwFmct1uCDEDXxZivwDf7khbUlI/GDg13LXsHQbxRaZNotcju\nibG9DNwInlBDpJ/grjlz/KG/LCmYrQE5RAnuqxVe812pc2ndSkTOGvcds7n7Gir/\n1S6zn2d5g2KeYtaMEYV1jArjzsFIdZ4M2R0ZTAsArcJy2ZGZ655j54Df7yzviNpD\nTz6nQQv1DHEpdogys+rOUTXrVhSpnsTacwztp/lvQsZl231THlCJcsySHRgMKmB+\nRKLLMfDfIaGeiZWRvEPEdurMWYkwWdYz9d+iEo3YTpWKy2QeCOEFZKMX5B2MXkdd\nNA==\n-----END CERTIFICATE-----\n"),
		},
	}
	client := fake.NewClientBuilder().WithScheme(schemeForClient).WithObjects(&sampleCert, &sampleSecret).Build()
	captureDir, err := os.MkdirTemp("", "testcaptureforcaCrt.json")
	assert.NoError(t, err)
	t.Log(captureDir)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-ca-crt-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err = captureCaCrtExpirationInfo(client, certificateListForTest, "cattle-system", captureDir, rc)
	assert.NoError(t, err)
}

// TestRedactHostNamesForCertificates tests the captureCertificates function
// GIVEN when sample cert with DNSNames and IPAddresses which are sensitive information
// WHEN captureCertificates is called on certain namespace
// THEN it should obfuscate the known hostnames with hashed value
// AND the output certificates.json file should NOT contain any of the sensitive information from KnownHostNames
func TestRedactHostNamesForCertificates(t *testing.T) {
	sampleCert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-certificate",
			Namespace: "cattle-system",
		},
		Spec: certmanagerv1.CertificateSpec{
			DNSNames:    []string{"example.com", "www.example.com", "api.example.com"},
			IPAddresses: []string{dummyIP1, dummyIP2},
		},
	}

	schemeForClient := k8scheme.Scheme
	err := certmanagerv1.AddToScheme(schemeForClient)
	assert.NoError(t, err)
	client := fake.NewClientBuilder().WithScheme(schemeForClient).WithObjects(sampleCert).Build()
	captureDir, err := os.MkdirTemp("", "testcaptureforcertificates")
	assert.NoError(t, err)
	t.Log(captureDir)
	defer cleanupTempDir(t, captureDir)
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	tempFile, err := os.CreateTemp(captureDir, "temporary-log-file-for-test")
	assert.NoError(t, err)
	SetMultiWriterOut(buf, tempFile)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	err = captureCertificates(client, common.CattleSystem, captureDir, rc)
	assert.NoError(t, err)

	// Check if the file is Sanitized as expected
	certLocation := filepath.Join(captureDir, common.CattleSystem, constants.CertificatesJSON)
	f, err := os.ReadFile(certLocation)
	assert.NoError(t, err, "Should not error reading certificates.json file")
	for k := range KnownHostNames {
		keyMatch, err := regexp.Match(k, f)
		assert.NoError(t, err, "Error while regex matching")
		assert.Falsef(t, keyMatch, "%s should be obfuscated from certificates.json file %s", k, string(f))
	}
}

// TestCreateParentsIfNecessary tests that parent directories are only created when intended
// GIVEN a temporary directory and a string representing a filePath  ,
// WHEN I call a function to create a parent directory
// THEN expect it to only create the parent directories if it does not exist and if the file path contains a "/"
func TestCreateParentsIfNecessary(t *testing.T) {
	err := os.Mkdir(constants.TestDirectory, 0700)
	defer os.RemoveAll(constants.TestDirectory)
	assert.Nil(t, err)
	createParentsIfNecessary(constants.TestDirectory, "files.txt")
	createParentsIfNecessary(constants.TestDirectory, "cluster-snapshot/files.txt")
	_, err = os.Stat(constants.TestDirectory + "/files.txt")
	assert.NotNil(t, err)
	_, err = os.Stat(constants.TestDirectory + "/cluster-snapshot")
	assert.Nil(t, err)

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

// returnBytesFromAFile is a helper function that returns the bytes of a file
func returnBytesFromAFile(clusterPath string) ([]byte, error) {
	var byteArray = []byte{}
	// Parse the json into local struct
	file, err := os.Open(clusterPath)
	if os.IsNotExist(err) {
		// The file may not exist if the component is not installed.
		return byteArray, nil
	}
	if err != nil {
		return byteArray, fmt.Errorf("failed to open file %s from cluster snapshot: %s", clusterPath, err.Error())
	}
	defer file.Close()

	byteArray, err = io.ReadAll(file)
	if err != nil {
		return byteArray, fmt.Errorf("Failed reading Json file %s: %s", clusterPath, err.Error())
	}

	return byteArray, nil
}
