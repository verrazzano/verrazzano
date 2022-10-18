// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	SECRET               = "testdata/secret.yaml"
	SECRET_BAD_NAMESPACE = "testdata/secret_bad_namespace.yaml"
	SECRET_INVALID       = "testdata/secret_invalid.yaml"
	SECRET_NO_NAMESPACE  = "testdata/secret_no_namespace.yaml"
)

// TestFindTestDataFile tests the FindTestDataFile function
// Given a filename, it should verify that the file exists
// and return the path to the file relative to pwd
func TestFindTestDataFile(t *testing.T) {
	asserts := assert.New(t)

	// File doesn't exist, should return an error
	filename := "test-file"

	_, err := FindTestDataFile(filename)
	asserts.Error(err)
	asserts.EqualError(err, fmt.Sprintf("failed to find test data file: %s", filename))

	// File exists, should find the file
	filename = SECRET
	file, err := FindTestDataFile(filename)

	asserts.NoError(err)
	asserts.Equal(filename, file)
}

// TestCreateOrUpdateResourceFromFile tests the CreateOrUpdateResourceFromFile function
// Given a yaml file, create the resource
func TestCreateOrUpdateResourceFromFile(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromFile(file)
	asserts.NoError(err)

	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromBytes tests the CreateOrUpdateResourceFromBytes function
// Given a stream of bytes, create the resource
func TestCreateOrUpdateResourceFromBytes(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	bytes, err := os.ReadFile(file)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	err = createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromBytes(bytes)
	asserts.NoError(err)

	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromFileInCluster tests the CreateOrUpdateResourceFromFileInCluster function
// Given a yaml file and the kubeconfig path, create the resource in the namespace
// Given a yaml file with bad namespace and the kubeconfig path, return an error
// Given a yaml file with invalid namespace and the kubeconfig path, return an error
func TestCreateOrUpdateResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Creating a resource with a valid yaml file
	// and in a namespace that exists
	// should not return an error
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.NoError(err)

	// Creating a resource in a namespace that doesn't exist
	// should return an error
	file = SECRET_BAD_NAMESPACE
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	// Passing a yaml file with no specified namespace
	// should return an error
	file = SECRET_NO_NAMESPACE
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	// Passing an invalid yaml file to create a resource
	// should return an error
	file = SECRET_INVALID
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromFileInGeneratedNamespace tests the
// CreateOrUpdateResourceFromFileInGeneratedNamespace
// Given a yaml file, create the resource in the provided namespace
func TestCreateOrUpdateResourceFromFileInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromFileInGeneratedNamespace(file, "default")
	asserts.NoError(err)

	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromFileInClusterInGeneratedNamespace tests
// the CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace function
// Given a yaml file with no namespace and the kubeconfig path, create the resource in the provided namespace
// When provided with a bad namespace, return an error
// Given an invalid yaml file and the kubeconfig path, return an error
func TestCreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET_NO_NAMESPACE

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.NoError(err)

	// Namespace doesn't exist, should return an error
	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "test")
	asserts.Error(err)

	file = SECRET_INVALID
	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.Error(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestDeleteResourceFromFile tests the DeleteResourceFromFile
// Given a yaml file, delete the resource
func TestDeleteResourceFromFile(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = DeleteResourceFromFile(file)
	asserts.NoError(err)

	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestDeleteResourceFromFileInCluster tests the DeleteResourceFromFileInCluster function
// Given a yaml and the kubeconfig path, delete the resource
// Given a yaml with bad namespace and the kubeconfig path, return an error
// Given an invalid yaml and the kubeconfig path, return an error
func TestDeleteResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Resource not found error is not returned, so
	// check for no error
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.NoError(err)

	file = SECRET_BAD_NAMESPACE
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	file = SECRET_INVALID
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestDeleteResourceFromFileInClusterInGeneratedNamespace tests
// the DeleteResourceFromFileInClusterInGeneratedNamespace function
// Given a yaml with no namespace, delete the resource in the provided namespace
// When provided with a bad namespace, return an error
// Given an invalid yaml file, return an error
func TestDeleteResourceFromFileInClusterInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET_NO_NAMESPACE

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.NoError(err)

	// Namespace doesn't exist, expect an error
	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "test")
	asserts.Error(err)

	file = SECRET_INVALID
	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.Error(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// TestPatchResourceFromFileInCluster tests PatchResourceFromFileInCluster function
// Given a yaml file, patch the resource if it exists
// Given an invalid yaml file, return an error
func TestPatchResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := SECRET

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: ""}

	// Patching a resource that doesn't exist should return an error
	err = PatchResourceFromFileInCluster(gvr, "default", "test-secret", file, kubeConfigPath)
	asserts.Error(err)

	file = SECRET_INVALID
	err = PatchResourceFromFileInCluster(gvr, "default", "test-secret", file, kubeConfigPath)
	asserts.Error(err)

	err = deleteFakeKubeConfig()
	asserts.NoError(err)
}

// newServer returns a httptest server which the
// dynamic client and discovery client can send
// GET/POST/DELETE requests to instead of a real cluster
func newServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
		case "/api/v1/namespaces/default":
			obj = &metav1.APIVersions{
				TypeMeta: metav1.TypeMeta{
					Kind: "APIVersions",
				},
				Versions: []string{
					"v1",
				},
			}
		case "/api":
			obj = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/api/v1":
			obj = &metav1.APIResourceList{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{Name: "secrets", Namespaced: true, Kind: "Secret"},
				},
			}
		case "/api/v1/namespaces/default/secrets":
			// POST request, return the raw request body
			body, _ := io.ReadAll(req.Body)
			w.Write(body)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(obj)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))
	return server
}

// createFakeKubeConfig creates a fake kubeconfig
// in the pwd with the url of the httptest server
func createFakeKubeConfig(url string) error {
	fakeKubeConfig, err := os.Create("dummy-kubeconfig")
	defer fakeKubeConfig.Close()

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(fakeKubeConfig, `apiVersion: v1
clusters:
- cluster:
    # This is dummy data
    certificate-authority-data: RFVNTVkgQ0VSVElGSUNBVEU=
    server: %s
  name: user-test
users:
- name: user-test
contexts:
- context:
    cluster: user-test
    user: user-test
  name: user-test
current-context: user-test`, url)

	return err
}

func getFakeKubeConfigPath() (string, error) {
	pwd, err := os.Getwd()

	if err != nil {
		return pwd, err
	}

	pwd = pwd + "/dummy-kubeconfig"
	return pwd, nil
}

func deleteFakeKubeConfig() error {
	err := os.Remove("dummy-kubeconfig")
	return err
}
