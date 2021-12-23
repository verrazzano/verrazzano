// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetAppConfig returns the specified appconfig in a given namespace
func GetAppConfig(namespace string, name string) (*unstructured.Unstructured, error) {
	client, err := GetDynamicClient()
	if err != nil {
		return nil, err
	}
	a, err := client.Resource(getOamAppConfigScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return a, nil
}

// getOamAppConfigScheme returns the appconfig scheme needed to get unstructured data
func getOamAppConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "core.oam.dev",
		Version:  "v1alpha2",
		Resource: "ApplicationConfiguration",
	}
}
