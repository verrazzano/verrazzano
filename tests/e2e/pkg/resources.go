// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

var nsGvr = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

// CreateOrUpdateResourceFromFile creates or updates a Kubernetes resources from a YAML test data file.
// The test data file is found using the FindTestDataFile function.
// This is intended to be equivalent to `kubectl apply`
// The cluster used is the one set by default in the environment
func CreateOrUpdateResourceFromFile(file string) error {
	return CreateOrUpdateResourceFromFileInCluster(file, GetKubeConfigPathFromEnv())
}

// CreateOrUpdateResourceFromFileInCluster is identical to CreateOrUpdateResourceFromFile, except that
// it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func CreateOrUpdateResourceFromFileInCluster(file string, kubeconfigPath string) error {
	found, err := FindTestDataFile(file)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	Log(Info, fmt.Sprintf("Found resource: %s", found))
	return createOrUpdateResourceFromBytes(bytes, GetKubeConfigGivenPath(kubeconfigPath))
}

// createOrUpdateResourceFromBytes creates or updates a Kubernetes resource from bytes.
// This is intended to be equivalent to `kubectl apply`
func createOrUpdateResourceFromBytes(data []byte, config *rest.Config) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))

	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		// Unmarshall the YAML bytes into an Unstructured.
		uns := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns)
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			// all resources must have been read
			return nil
		}

		// Attempt to create the resource.
		_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Create(context.TODO(), uns, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			// Get, read the resource version, and then update the resource.
			resource, err := client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Get(context.TODO(), uns.GetName(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get resource for update: %w", err)
			}
			uns.SetResourceVersion(resource.GetResourceVersion())
			_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Update(context.TODO(), uns, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	}
	// no return since you can't get here
}

func readNextResourceFromBytes(reader *utilyaml.YAMLReader, mapper *restmapper.DeferredDiscoveryRESTMapper, client dynamic.Interface, uns *unstructured.Unstructured) (*meta.RESTMapping, error) {
	// Read one section of the YAML
	buf, err := reader.Read()
	// Return success if the whole file has been read.
	if err == io.EOF {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read resource section: %w", err)
	}

	if err = yaml.Unmarshal(buf, &uns.Object); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	// Check to make sure the namespace of the resource exists.
	_, err = client.Resource(nsGvr).Get(context.TODO(), uns.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find resource namespace: %w", err)
	}

	// Map the object's GVK to a GVR
	unsGvk := schema.FromAPIVersionAndKind(uns.GetAPIVersion(), uns.GetKind())
	unsMap, err := mapper.RESTMapping(unsGvk.GroupKind(), unsGvk.Version)
	if err != nil {
		return unsMap, fmt.Errorf("failed to map resource kind: %w", err)
	}
	return unsMap, nil
}

// DeleteResourceFromFile deletes Kubernetes resources using names found in a YAML test data file.
// This is intended to be equivalent to `kubectl delete`
// The test data file is found using the FindTestDataFile function.
func DeleteResourceFromFile(file string) error {
	return DeleteResourceFromFileInCluster(file, GetKubeConfigPathFromEnv())
}

// DeleteResourceFromFileInCluster is identical to DeleteResourceFromFile, except that
// // it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func DeleteResourceFromFileInCluster(file string, kubeconfigPath string) error {
	found, err := FindTestDataFile(file)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	return deleteResourceFromBytes(bytes, kubeconfigPath)
}

// deleteResourceFromBytes deletes Kubernetes resources using names found in YAML bytes.
// This is intended to be equivalent to `kubectl delete`
func deleteResourceFromBytes(data []byte, kubeconfigPath string) error {
	config := GetKubeConfigGivenPath(kubeconfigPath)
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))

	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		// Unmarshall the YAML bytes into an Unstructured.
		uns := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns)
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			// all resources must have been read
			return nil
		}

		// Delete the resource.
		err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Delete(context.TODO(), uns.GetName(), metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			fmt.Printf("Failed to delete %s/%v", uns.GetNamespace(), uns.GroupVersionKind())
		}
	}
}

// PatchResourceFromFileInCluster patches a Kubernetes resource from a given patch file in the specified cluster
// If the given patch file has a ".yaml" extension, the contents will be converted to JSON
// This is intended to be equivalent to `kubectl patch`
func PatchResourceFromFileInCluster(gvr schema.GroupVersionResource, namespace string, name string, patchFile string, kubeconfigPath string) error {
	found, err := FindTestDataFile(patchFile)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	patchBytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	Log(Info, fmt.Sprintf("Found resource: %s", found))

	if strings.HasSuffix(patchFile, ".yaml") {
		patchBytes, err = utilyaml.ToJSON(patchBytes)
		if err != nil {
			return fmt.Errorf("Could not convert patch data to JSON: %w", err)
		}
	}

	return patchResourceFromBytes(gvr, namespace, name, patchBytes, GetKubeConfigGivenPath(kubeconfigPath))
}

// patchResourceFromBytes patches a Kubernetes resource from bytes. The contents of the byte slice must be in
// JSON format. This is intended to be equivalent to `kubectl patch`.
func patchResourceFromBytes(gvr schema.GroupVersionResource, namespace string, name string, patchDataJSON []byte, config *rest.Config) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Attempt to patch the resource.
	_, err = client.Resource(gvr).Namespace(namespace).Patch(context.TODO(), name, types.MergePatchType, patchDataJSON, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to patch %s/%v: %w", namespace, gvr, err)
	}
	return nil
}
