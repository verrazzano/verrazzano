// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Secret             = "testdata/secret.yaml"
	SecretBadNamespace = "testdata/secret_bad_namespace.yaml"
	SecretInvalid      = "testdata/secret_invalid.yaml"
	SecretNoNamespace  = "testdata/secret_no_namespace.yaml"
)

// TestCreateOrUpdateResourceFromFile tests the CreateOrUpdateResourceFromFile function
// Given a yaml file, create the resource
func TestCreateOrUpdateResourceFromFile(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	defer os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	logger := vzlog.DefaultLogger().GetZapLogger()
	err = CreateOrUpdateResourceFromFile(file, logger)
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromBytes tests the CreateOrUpdateResourceFromBytes function
// Given a stream of bytes, create the resource
func TestCreateOrUpdateResourceFromBytes(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	bytes, err := os.ReadFile(file)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	err = createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	defer os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromBytes(bytes, vzlog.DefaultLogger().GetZapLogger())
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromFileInCluster tests the CreateOrUpdateResourceFromFileInCluster function
// Given a yaml file and the kubeconfig path, create the resource in the namespace
// Given a yaml file with bad namespace and the kubeconfig path, return an error
// Given a yaml file with invalid namespace and the kubeconfig path, return an error
func TestCreateOrUpdateResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
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
	file = SecretBadNamespace
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	// Passing a yaml file with no specified namespace
	// should return an error
	file = SecretNoNamespace
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	// Passing an invalid yaml file to create a resource
	// should return an error
	file = SecretInvalid
	err = CreateOrUpdateResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)
}

// TestCreateOrUpdateResourceFromFileInGeneratedNamespace tests the
// CreateOrUpdateResourceFromFileInGeneratedNamespace
// Given a yaml file, create the resource in the provided namespace
func TestCreateOrUpdateResourceFromFileInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	defer os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromFileInGeneratedNamespace(file, "default")
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromFileInClusterInGeneratedNamespace tests
// the CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace function
// Given a yaml file with no namespace and the kubeconfig path, create the resource in the provided namespace
// When provided with a bad namespace, return an error
// Given an invalid yaml file and the kubeconfig path, return an error
func TestCreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SecretNoNamespace

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.NoError(err)

	// Namespace doesn't exist, should return an error
	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "test")
	asserts.Error(err)

	file = SecretInvalid
	err = CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.Error(err)
}

// TestDeleteResourceFromFile tests the DeleteResourceFromFile
// Given a yaml file, delete the resource
func TestDeleteResourceFromFile(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	defer os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = DeleteResourceFromFile(file, vzlog.DefaultLogger().GetZapLogger())
	asserts.NoError(err)
}

// TestDeleteResourceFromFileInCluster tests the DeleteResourceFromFileInCluster function
// Given a yaml and the kubeconfig path, delete the resource
// Given a yaml with bad namespace and the kubeconfig path, return an error
// Given an invalid yaml and the kubeconfig path, return an error
func TestDeleteResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Resource not found error is not returned, so
	// check for no error
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.NoError(err)

	file = SecretBadNamespace
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)

	file = SecretInvalid
	err = DeleteResourceFromFileInCluster(file, kubeConfigPath)
	asserts.Error(err)
}

// TestDeleteResourceFromFileInGeneratedNamespace tests
// the DeleteResourceFromFileInGeneratedNamespace
// Given a yaml with no namespace, delete the resource
func TestDeleteResourceFromFileInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SecretNoNamespace

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	defer os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)

	// Test using environment variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, kubeConfigPath)
	asserts.NoError(err)

	err = DeleteResourceFromFileInGeneratedNamespace(file, "default")
	asserts.NoError(err)
}

// TestDeleteResourceFromFileInClusterInGeneratedNamespace tests
// the DeleteResourceFromFileInClusterInGeneratedNamespace function
// Given a yaml with no namespace, delete the resource in the provided namespace
// When provided with a bad namespace, return an error
// Given an invalid yaml file, return an error
func TestDeleteResourceFromFileInClusterInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)
	file := SecretNoNamespace

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.NoError(err)

	// Namespace doesn't exist, expect an error
	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "test")
	asserts.Error(err)

	file = SecretInvalid
	err = DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeConfigPath, "default")
	asserts.Error(err)
}

// TestPatchResourceFromFileInCluster tests PatchResourceFromFileInCluster function
// Given a yaml file, patch the resource if it exists
// Given an invalid yaml file, return an error
func TestPatchResourceFromFileInCluster(t *testing.T) {
	asserts := assert.New(t)
	file := Secret

	server := newServer()
	defer server.Close()

	err := createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: ""}

	// Patching a resource that doesn't exist should return an error
	err = PatchResourceFromFileInCluster(gvr, "default", "test-secret", file, kubeConfigPath)
	asserts.Error(err)

	file = SecretInvalid
	err = PatchResourceFromFileInCluster(gvr, "default", "test-secret", file, kubeConfigPath)
	asserts.Error(err)
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
