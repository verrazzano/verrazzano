// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
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
func CreateOrUpdateResourceFromFile(file string, log *zap.SugaredLogger) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}

	return CreateOrUpdateResourceFromFileInCluster(file, kubeconfigPath)
}

// CreateOrUpdateResourceFromBytes creates or updates a Kubernetes resources from a YAML test data byte array.
// The cluster used is the one set by default in the environment
func CreateOrUpdateResourceFromBytes(data []byte, log *zap.SugaredLogger) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}

	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	return CreateOrUpdateResourceFromBytesUsingConfig(data, config)
}

// CreateOrUpdateResourceFromFileInCluster is identical to CreateOrUpdateResourceFromFile, except that
// it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func CreateOrUpdateResourceFromFileInCluster(file string, kubeconfigPath string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}

	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}
	return CreateOrUpdateResourceFromBytesUsingConfig(bytes, config)
}

// CreateOrUpdateResourceFromBytesUsingConfig creates or updates a Kubernetes resource from bytes.
// This is intended to be equivalent to `kubectl apply`
func CreateOrUpdateResourceFromBytesUsingConfig(data []byte, config *rest.Config) error {
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
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns, "")
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			// all resources must have been read
			return nil
		}

		namespace := uns.GetNamespace()

		if namespace != "" {
			// Attempt to create the resource.
			_, err = client.Resource(unsMap.Resource).Namespace(namespace).Create(context.TODO(), uns, metav1.CreateOptions{})
			if err != nil && errors.IsAlreadyExists(err) {
				// Get, read the resource version, and then update the resource.
				resource, err := client.Resource(unsMap.Resource).Namespace(namespace).Get(context.TODO(), uns.GetName(), metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to get resource for update: %w", err)
				}
				uns.SetResourceVersion(resource.GetResourceVersion())
				_, err = client.Resource(unsMap.Resource).Namespace(namespace).Update(context.TODO(), uns, metav1.UpdateOptions{})
				if err != nil {
					return fmt.Errorf("failed to update resource: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to create resource: %w", err)
			}
		} else {
			_, err = client.Resource(unsMap.Resource).Create(context.TODO(), uns, metav1.CreateOptions{})
			if err != nil && errors.IsAlreadyExists(err) {
				resource, err := client.Resource(unsMap.Resource).Get(context.TODO(), uns.GetName(), metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to get resource for update: %v", err)
				}
				uns.SetResourceVersion(resource.GetResourceVersion())
				_, err = client.Resource(unsMap.Resource).Update(context.TODO(), uns, metav1.UpdateOptions{})
				if err != nil {
					return fmt.Errorf("failed to update resource: %v", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to create resource: %v", err)
			}
		}
	}
	// no return since you can't get here
}

// CreateOrUpdateResourceFromFileInGeneratedNamespace creates or updates a Kubernetes resources from a YAML test data file.
// The test data file is found using the FindTestDataFile function.
// Namespaces are not in the resource yaml files. They are generated and passed in
// Resources will be created in the namespace that is passed in
// This is intended to be equivalent to `kubectl apply`
// The cluster used is the one set by default in the environment
func CreateOrUpdateResourceFromFileInGeneratedNamespace(file string, namespace string) error {
	var logger = vzlog.DefaultLogger()
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		logger.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}

	return CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, namespace)
}

// CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace is identical to CreateOrUpdateResourceFromFileInGeneratedNamespace, except that
// it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(file string, kubeconfigPath string, namespace string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}

	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	return createOrUpdateResourceFromBytesInGeneratedNamespace(bytes, config, namespace)
}

// createOrUpdateResourceFromBytesInGeneratedNamespace creates or updates a Kubernetes resource from bytes in the provided namespace.
// This is intended to be equivalent to `kubectl apply`
func createOrUpdateResourceFromBytesInGeneratedNamespace(data []byte, config *rest.Config, namespace string) error {
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
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns, namespace)
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			// all resources must have been read
			return nil
		}
		uns.SetNamespace(namespace)

		// Attempt to create the resource.
		_, err = client.Resource(unsMap.Resource).Namespace(namespace).Create(context.TODO(), uns, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			// Get, read the resource version, and then update the resource.
			resource, err := client.Resource(unsMap.Resource).Namespace(namespace).Get(context.TODO(), uns.GetName(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get resource for update: %w", err)
			}
			uns.SetResourceVersion(resource.GetResourceVersion())
			_, err = client.Resource(unsMap.Resource).Namespace(namespace).Update(context.TODO(), uns, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	}
	// no return since you can't get here
}

func readNextResourceFromBytes(reader *utilyaml.YAMLReader, mapper *restmapper.DeferredDiscoveryRESTMapper, client dynamic.Interface, uns *unstructured.Unstructured, namespace string) (*meta.RESTMapping, error) {
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

	// If namespace is nil, then try to get it from uns
	if namespace == "" {
		namespace = uns.GetNamespace()
	}

	// If the resource has a namespace, check to make sure the namespace exists.
	if namespace != "" {
		_, err = client.Resource(nsGvr).Get(context.TODO(), namespace, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to find resource namespace: %w", err)
		}
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
func DeleteResourceFromFile(file string, log *zap.SugaredLogger) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}
	return DeleteResourceFromFileInCluster(file, kubeconfigPath)
}

// DeleteResourceFromFileInCluster is identical to DeleteResourceFromFile, except that
// // it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func DeleteResourceFromFileInCluster(file string, kubeconfigPath string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	return deleteResourceFromBytes(bytes, config)
}

// deleteResourceFromBytes deletes Kubernetes resources using names found in YAML bytes.
// This is intended to be equivalent to `kubectl delete`
func deleteResourceFromBytes(data []byte, config *rest.Config) error {
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
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns, "")
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

// DeleteResourceFromFileInGeneratedNamespace deletes Kubernetes resources using names found in a YAML test data file.
// The test data file is found using the FindTestDataFile function.
// The namespace is generated and passed in
func DeleteResourceFromFileInGeneratedNamespace(file string, namespace string) error {
	var logger = vzlog.DefaultLogger()
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		logger.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}
	return DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, namespace)
}

// DeleteResourceFromFileInClusterInGeneratedNamespace is identical to DeleteResourceFromFileInGeneratedNamespace,
// except that it uses the cluster specified by the kubeconfigPath argument instead of the default cluster in the environment
func DeleteResourceFromFileInClusterInGeneratedNamespace(file string, kubeconfigPath string, namespace string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	return deleteResourceFromBytesInGeneratedNamespace(bytes, config, namespace)
}

// deleteResourceFromBytes deletes Kubernetes resources using names found in YAML bytes.
// This is intended to be equivalent to `kubectl delete`
func deleteResourceFromBytesInGeneratedNamespace(data []byte, config *rest.Config, namespace string) error {
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
		unsMap, err := readNextResourceFromBytes(reader, mapper, client, uns, namespace)
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			// all resources must have been read
			return nil
		}

		// Delete the resource.
		err = client.Resource(unsMap.Resource).Namespace(namespace).Delete(context.TODO(), uns.GetName(), metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			fmt.Printf("Failed to delete %s/%v", namespace, uns.GroupVersionKind())
		}
	}
}

// PatchResourceFromFileInCluster patches a Kubernetes resource from a given patch file in the specified cluster
// If the given patch file has a ".yaml" extension, the contents will be converted to JSON
// This is intended to be equivalent to `kubectl patch`
func PatchResourceFromFileInCluster(gvr schema.GroupVersionResource, namespace string, name string, patchFile string, kubeconfigPath string) error {
	patchBytes, err := os.ReadFile(patchFile)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}

	if strings.HasSuffix(patchFile, ".yaml") {
		patchBytes, err = utilyaml.ToJSON(patchBytes)
		if err != nil {
			return fmt.Errorf("could not convert patch data to JSON: %w", err)
		}
	}

	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	return PatchResourceFromBytes(gvr, types.MergePatchType, namespace, name, patchBytes, config)
}

// PatchResourceFromBytes patches a Kubernetes resource from bytes. The contents of the byte slice must be in
// JSON format. This is intended to be equivalent to `kubectl patch`.
func PatchResourceFromBytes(gvr schema.GroupVersionResource, patchType types.PatchType, namespace string, name string, patchDataJSON []byte, config *rest.Config) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Attempt to patch the resource.
	_, err = client.Resource(gvr).Namespace(namespace).Patch(context.TODO(), name, patchType, patchDataJSON, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch %s/%v: %w", namespace, gvr, err)
	}
	return nil
}
