// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newUnstructured(apiVersion, kind, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

type FakeDynamicClient struct {
	scheme *runtime.Scheme
	server *httptest.Server
}

func (f *FakeDynamicClient) GetDynamicClient() (dynamic.Interface, error) {

	client := fake.NewSimpleDynamicClient(f.scheme, newUnstructured("v1", "Namespace", "default"))
	return client, nil
}

func (f *FakeDynamicClient) GetDiscoveryClient() (*discovery.DiscoveryClient, error) {
	return discovery.NewDiscoveryClientForConfig(&rest.Config{Host: f.server.URL})
}

// TestFindTestDataFile
// Given a filename, it should verify that the file exists
// and return the path to the file relative to pwd
func TestFindTestDataFile(t *testing.T) {
	asserts := assert.New(t)

	// File doesn't exist, should return an error
	filename := "testfile"

	_, err := FindTestDataFile(filename)
	asserts.Error(err)
	asserts.EqualError(err, fmt.Sprintf("failed to find test data file: %s", filename))

	// File exists, should find the file
	filename = "testdata/secret_create.yaml"
	file, err := FindTestDataFile(filename)

	asserts.NoError(err)
	asserts.Equal(filename, file)
}

func TestCreateOrUpdateResourceFromFile(t *testing.T) {
	asserts := assert.New(t)

	filename := "testdata/secret_create.yaml"

	// Find the file
	found, err := FindTestDataFile(filename)
	asserts.NoError(err)

	// Read the file
	bytes, err := os.ReadFile(found)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	dc := &FakeDynamicClient{scheme: runtime.NewScheme(), server: server}

	// Create a secret resource from yaml
	err = createOrUpdateResourceFromBytes(bytes, dc)
	asserts.NoError(err)
}

// TestCreateOrUpdateResourceFromBytesInGeneratedNamespace
// Given a yaml file and a namespace string it should create the resource
// in the provided namespace
func TestCreateOrUpdateResourceFromBytesInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)

	filename := "testdata/secret_create.yaml"

	// Find the file
	found, err := FindTestDataFile(filename)
	asserts.NoError(err)

	// Read the file
	bytes, err := os.ReadFile(found)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	dc := &FakeDynamicClient{scheme: runtime.NewScheme(), server: server}

	err = createOrUpdateResourceFromBytesInGeneratedNamespace(bytes, dc, "default")
	asserts.NoError(err)
}

// TestDeleteResourceFromBytes
// Given a yaml try and delete the resource
func TestDeleteResourceFromBytes(t *testing.T) {
	asserts := assert.New(t)

	filename := "testdata/secret_create.yaml"

	found, err := FindTestDataFile(filename)
	asserts.NoError(err)

	bytes, err := os.ReadFile(found)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	dc := &FakeDynamicClient{scheme: runtime.NewScheme(), server: server}

	// Try and delete the secret resource from yaml
	// Since the resource doesn't exist
	// This gives resource not found error
	// But that error is not returned, so we check for no error
	err = deleteResourceFromBytes(bytes, dc)
	asserts.NoError(err)

	// Create the resource and then delete it
	err = createOrUpdateResourceFromBytes(bytes, dc)
	asserts.NoError(err)

	err = deleteResourceFromBytes(bytes, dc)
	asserts.NoError(err)
}

// TestDeleteResourceFromBytesInGeneratedNamespace
// Given a yaml try and delete the resource in the provided namespace
func TestDeleteResourceFromBytesInGeneratedNamespace(t *testing.T) {
	asserts := assert.New(t)

	filename := "testdata/secret_create.yaml"

	found, err := FindTestDataFile(filename)
	asserts.NoError(err)

	bytes, err := os.ReadFile(found)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	dc := &FakeDynamicClient{scheme: runtime.NewScheme(), server: server}

	// Try and delete the secret resource from yaml
	// Since the resource doesn't exist
	// This gives resource not found error
	// But that error is not returned, so we check for no error
	err = deleteResourceFromBytesInGeneratedNamespace(bytes, dc, "default")
	asserts.NoError(err)
}

// TestPatchResourceFromBytes
// patch the file which doesn't exist
// should return an error
func TestPatchResourceFromBytes(t *testing.T) {
	asserts := assert.New(t)

	filename := "testdata/secret_create.yaml"

	found, err := FindTestDataFile(filename)
	asserts.NoError(err)

	bytes, err := os.ReadFile(found)
	asserts.NoError(err)

	server := newServer()
	defer server.Close()

	dc := &FakeDynamicClient{scheme: runtime.NewScheme(), server: server}

	bytes, err = utilyaml.ToJSON(bytes)
	asserts.NoError(err)

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}

	err = patchResourceFromBytes(gvr, "default", "test-secret", bytes, dc)
	asserts.EqualError(err, "failed to patch default//v1, Resource=secrets: secrets \"test-secret\" not found")
}

// newServer return a httptest server which the
// fake dynamic client and discovery client can connect/send requests to
// instead of a real live cluster
func newServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
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
