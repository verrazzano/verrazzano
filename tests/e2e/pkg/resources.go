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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
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
// This is indented to be equivalent to `kubectl apply`
func CreateOrUpdateResourceFromFile(file string) error {
	found, err := FindTestDataFile(file)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	Log(Info, fmt.Sprintf("Found resource: %s", found))
	return CreateOrUpdateResourceFromBytes(bytes)
}

// CreateOrUpdateResourceFromBytes creates or updates a Kubernetes resource from bytes.
// This is indented to be equivalent to `kubectl apply`
func CreateOrUpdateResourceFromBytes(data []byte) error {
	config := GetKubeConfig()
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
		// Read one section of the YAML
		buf, err := reader.Read()
		// Return success if the whole file has been read.
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read resource section: %w", err)
		}

		// Unmarshall the YAML bytes into an Unstructured.
		uns := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}
		if err = yaml.Unmarshal(buf, &uns.Object); err != nil {
			return fmt.Errorf("failed to unmarshal resource: %w", err)
		}

		// Check to make sure the namespace of the resource exists.
		_, err = client.Resource(nsGvr).Get(context.TODO(), uns.GetNamespace(), metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to find resource namespace: %w", err)
		}

		// Map the object's GVK to a GVR
		unsGvk := schema.FromAPIVersionAndKind(uns.GetAPIVersion(), uns.GetKind())
		unsMap, err := mapper.RESTMapping(unsGvk.GroupKind(), unsGvk.Version)
		if err != nil {
			return fmt.Errorf("failed to map resource kind: %w", err)
		}

		// Attempt to create the resource.
		_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Create(context.TODO(), uns, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			// Update the resource.
			_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Update(context.TODO(), uns, metav1.UpdateOptions{})
		}
		if err != nil {
			return fmt.Errorf("failed to create or update resource: %w", err)
		}
	}
	return nil
}

// DeleteResourceFromFile deletes Kubernetes resources using names found in a YAML test data file.
// This is indented to be equivalent to `kubectl delete`
// The test data file is found using the FindTestDataFile function.
func DeleteResourceFromFile(file string) error {
	found, err := FindTestDataFile(file)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	return DeleteResourceFromBytes(bytes)
}

// DeleteResourceFromBytes deletes Kubernetes resources using names found in YAML bytes.
// This is indented to be equivalent to `kubectl delete`
func DeleteResourceFromBytes(data []byte) error {
	config := GetKubeConfig()
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
		// Read one section of the YAML
		buf, err := reader.Read()
		// Return success if the whole file has been read.
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read resource section: %w", err)
		}

		// Unmarshall the YAML bytes into an Unstructured.
		uns := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}
		if err = yaml.Unmarshal(buf, &uns.Object); err != nil {
			return fmt.Errorf("failed to unmarshal resource: %w", err)
		}

		// Map the object's GVK to a GVR
		unsGvk := schema.FromAPIVersionAndKind(uns.GetAPIVersion(), uns.GetKind())
		unsMap, err := mapper.RESTMapping(unsGvk.GroupKind(), unsGvk.Version)
		if err != nil {
			return fmt.Errorf("failed to map resource kind: %w", err)
		}

		// Delete the resource.
		err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Delete(context.TODO(), uns.GetName(), metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			fmt.Printf("Failed to delete %s/%v", uns.GetNamespace(), uns.GroupVersionKind())
		}
	}
}
