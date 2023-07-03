// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	addonsGroup                        = "addons.cluster.x-k8s.io"
	v1Beta1API                         = "v1beta1"
	clusterResourceSetBindingsResource = "clusterresourcesetbindings"
	clusterResourceSetBindingsKind     = "ClusterResourceSetBindings"
	clusterResourceSetsResource        = "clusterresourcesets"
	clusterResourceSetsKind            = "ClusterResourceSets"
)

func createGVR(group string, version string, resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
}

// captureCapiResources captures resources related to ClusterAPI
func captureCapiResources(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	if err := captureResource(dynamicClient, createGVR(addonsGroup, v1Beta1API, clusterResourceSetBindingsResource), clusterResourceSetBindingsKind, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	if err := captureResource(dynamicClient, createGVR(addonsGroup, v1Beta1API, clusterResourceSetsResource), clusterResourceSetsKind, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	return nil
}

func captureResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, kind string, namespace, captureDir string, vzHelper VZHelper) error {
	list, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		LogError(fmt.Sprintf("An error occurred while getting the %s in namespace %s: %s\n", kind, namespace, err.Error()))
	}
	if len(list.Items) > 0 {
		LogMessage(fmt.Sprintf("%s in namespace: %s ...\n", kind, namespace))
		if err = createFile(list, namespace, fmt.Sprintf("%s.json", strings.ToLower(kind)), captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}
